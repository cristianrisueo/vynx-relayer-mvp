package ingress

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// WebSocket timing constants.
// pingPeriod must be strictly less than pongWait to ensure pings are sent before
// the read deadline fires on the client side.
const (
	maxMessageSize = 4096                // maximum bytes per solver message
	pongWait       = 60 * time.Second    // idle read deadline per client
	pingPeriod     = (pongWait * 9) / 10 // keepalive ping interval (54 s)
	writeWait      = 10 * time.Second    // write deadline per frame
	sendBufSize    = 256                 // per-client outbound queue depth
)

// upgrader converts an HTTP connection to a WebSocket connection.
// CheckOrigin always returns true — in production, restrict this to known
// solver origin domains or validate a bearer token in the HTTP handshake.
var upgrader = websocket.Upgrader{ //nolint:gosec // CheckOrigin intentionally permissive in dev; restrict in production
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// client represents a single connected solver's WebSocket session.
type client struct {
	conn *websocket.Conn
	send chan []byte // buffered outbound queue; closed by Hub on unregister
}

// Hub manages all active solver WebSocket connections.
// It is safe for concurrent use — the clients map is protected by mu.
//
// Lock sharding: mu guards only the map pointer operations (register/unregister).
// Per-client write operations are serialised by the client's send channel and its
// dedicated writePump goroutine — no Hub-level lock is held during I/O.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
	bidCh   chan<- *core.Bid
	logger  *zap.Logger
}

// NewHub constructs a Hub that forwards parsed solver bids to bidCh.
func NewHub(bidCh chan<- *core.Bid, logger *zap.Logger) *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
		bidCh:   bidCh,
		logger:  logger,
	}
}

// Run blocks until ctx is cancelled, then performs a graceful shutdown by closing
// every connected client's send channel (which triggers writePump exit and then
// the underlying TCP connection close).
func (h *Hub) Run(ctx context.Context) {
	<-ctx.Done()
	h.logger.Info("mempool hub shutting down")

	h.mu.Lock()
	for c := range h.clients {
		close(c.send)
		delete(h.clients, c)
	}
	h.mu.Unlock()
}

// ServeWS upgrades an HTTP request to a WebSocket connection and registers the
// new solver client. Each client gets two goroutines: readPump and writePump.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	c := &client{
		conn: conn,
		send: make(chan []byte, sendBufSize),
	}

	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	h.logger.Info("solver connected", zap.String("remote_addr", r.RemoteAddr))

	go c.writePump(h)
	go c.readPump(h)
}

// BroadcastIntent serialises intent to JSON and delivers it to every connected
// solver's outbound queue. Slow consumers whose queue is full are disconnected
// immediately to prevent head-of-line blocking for fast solvers.
func (h *Hub) BroadcastIntent(intent *core.Intent) {
	msg, err := json.Marshal(intentBroadcast{
		Type: "new_intent",
		Intent: intentPayload{
			ID:           string(intent.ID),
			TokenIn:      intent.TokenIn.Hex(),
			TokenOut:     intent.TokenOut.Hex(),
			AmountIn:     intent.AmountIn.String(),
			MinAmountOut: intent.MinAmountOut.String(),
			Deadline:     intent.Deadline.Unix(),
		},
	})
	if err != nil {
		h.logger.Error("failed to marshal intent broadcast", zap.Error(err))
		return
	}

	// Collect slow clients without holding the write lock during unregister.
	var slow []*client

	h.mu.RLock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range slow {
		h.logger.Warn("solver too slow — disconnecting",
			zap.String("intent_id", string(intent.ID)),
		)
		h.unregisterClient(c)
	}
}

// unregisterClient removes c from the active clients map and closes its send channel.
// Idempotent: safe to call if c was already removed.
func (h *Hub) unregisterClient(c *client) {
	h.mu.Lock()
	if _, exists := h.clients[c]; exists {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// writePump serialises all outbound frames for one solver connection.
// It owns the write path for c.conn — only this goroutine calls conn.WriteMessage.
// A ticker sends WebSocket pings at pingPeriod to detect half-open connections.
func (c *client) writePump(h *Hub) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				// Hub closed the channel — send a clean close frame and exit.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump receives bid messages from a solver, parses them, and pushes the resulting
// core.Bid to the hub's bidCh. It is the only goroutine that calls conn.ReadMessage.
// When it exits (connection closed or read error), it unregisters the client.
func (c *client) readPump(h *Hub) {
	defer func() {
		h.unregisterClient(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warn("unexpected solver disconnect", zap.Error(err))
			}
			return
		}

		var req bidRequest
		if jsonErr := json.Unmarshal(msg, &req); jsonErr != nil {
			h.logger.Warn("invalid bid payload from solver", zap.Error(jsonErr))
			continue
		}

		amountOut, ok := new(big.Int).SetString(req.AmountOut, 10)
		if !ok || amountOut.Sign() <= 0 {
			h.logger.Warn("invalid bid amount_out", zap.String("raw", req.AmountOut))
			continue
		}

		gasPrice, ok := new(big.Int).SetString(req.GasPrice, 10)
		if !ok || gasPrice.Sign() <= 0 {
			h.logger.Warn("invalid bid gas_price", zap.String("raw", req.GasPrice))
			continue
		}

		bid := &core.Bid{
			IntentID:   core.IntentID(req.IntentID),
			Solver:     common.HexToAddress(req.Solver),
			AmountOut:  amountOut,
			GasPrice:   gasPrice,
			ReceivedAt: time.Now(),
		}

		// Non-blocking push: drop if the bid channel is saturated (HFT mandate).
		select {
		case h.bidCh <- bid:
		default:
			h.logger.Warn("bid channel full — dropping bid",
				zap.String("intent_id", req.IntentID),
			)
		}
	}
}

// ── Wire types ─────────────────────────────────────────────────────────────────

// bidRequest is the JSON shape expected from a solver over WebSocket.
type bidRequest struct {
	IntentID  string `json:"intent_id"`
	Solver    string `json:"solver"`
	AmountOut string `json:"amount_out"` // decimal wei string
	GasPrice  string `json:"gas_price"`  // decimal wei string
}

// intentBroadcast is the JSON shape pushed to all connected solvers when a new
// Intent is accepted by the relayer.
type intentBroadcast struct {
	Type   string        `json:"type"`
	Intent intentPayload `json:"intent"`
}

// intentPayload contains the subset of Intent fields solvers need to compute a bid.
type intentPayload struct {
	ID           string `json:"id"`
	TokenIn      string `json:"token_in"`
	TokenOut     string `json:"token_out"`
	AmountIn     string `json:"amount_in"`
	MinAmountOut string `json:"min_amount_out"`
	Deadline     int64  `json:"deadline"`
}
