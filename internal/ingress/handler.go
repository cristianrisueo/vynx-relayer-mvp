// Package ingress implements the HTTP REST and WebSocket ingress layer for the
// VynX Relayer. It is the sole external entry point for Intent submissions and
// Solver bid connections.
//
// Dependency rule: this package imports only internal/core. It must never import
// internal/auction or internal/txmanager — communication is via injected channels
// (Event Bus) wired in cmd/relayer/main.go.
package ingress

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// Handler serves the REST ingress endpoints for the VynX Relayer.
// Validated intents are forwarded to the auction engine via intentCh (Event Bus)
// and broadcast to connected solvers via the Hub.
type Handler struct {
	intentCh chan<- *core.Intent
	hub      *Hub
	logger   *zap.Logger
}

// NewHandler constructs a Handler wired to the provided Event Bus channels.
//
//   - intentCh: send-only end of the channel consumed by the auction dispatcher.
//   - hub:      WebSocket Hub used to broadcast accepted intents to solvers.
func NewHandler(intentCh chan<- *core.Intent, hub *Hub, logger *zap.Logger) *Handler {
	return &Handler{intentCh: intentCh, hub: hub, logger: logger}
}

// intentRequest is the JSON body for POST /v1/intent.
// AmountIn and MinAmountOut are decimal strings to preserve uint256 precision.
type intentRequest struct {
	ID           string `json:"id"`
	Sender       string `json:"sender"`
	TokenIn      string `json:"token_in"`
	TokenOut     string `json:"token_out"`
	AmountIn     string `json:"amount_in"`
	MinAmountOut string `json:"min_amount_out"`
	Deadline     int64  `json:"deadline"` // Unix timestamp (seconds)
	Nonce        uint64 `json:"nonce"`
	Signature    string `json:"signature"` // hex-encoded, optional in MVP
}

// intentResponse is the JSON body returned on 202 Accepted.
type intentResponse struct {
	IntentID string `json:"intent_id"`
	Status   string `json:"status"`
}

// SubmitIntent handles POST /v1/intent.
// On success it returns 202 Accepted immediately — the auction runs asynchronously.
// The handler is non-blocking: if the auction engine's channel is full it returns
// 503 so the caller can retry rather than stalling the ingress goroutine.
func (h *Handler) SubmitIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req intentRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	amountIn, ok := new(big.Int).SetString(req.AmountIn, 10)
	if !ok || amountIn.Sign() <= 0 {
		http.Error(w, "amount_in must be a positive decimal integer", http.StatusBadRequest)
		return
	}

	minAmountOut, ok := new(big.Int).SetString(req.MinAmountOut, 10)
	if !ok || minAmountOut.Sign() < 0 {
		http.Error(w, "min_amount_out must be a non-negative decimal integer", http.StatusBadRequest)
		return
	}

	deadline := time.Unix(req.Deadline, 0)
	if !deadline.After(time.Now()) {
		http.Error(w, "deadline must be in the future", http.StatusBadRequest)
		return
	}

	// Parse optional EIP-712 signature (not yet verified in MVP — added for forward-compat).
	var sig []byte
	if req.Signature != "" {
		var parseErr error
		sig, parseErr = hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
		if parseErr != nil {
			http.Error(w, "signature must be hex-encoded", http.StatusBadRequest)
			return
		}
	}

	intent := &core.Intent{
		ID:           core.IntentID(req.ID),
		Sender:       common.HexToAddress(req.Sender),
		TokenIn:      common.HexToAddress(req.TokenIn),
		TokenOut:     common.HexToAddress(req.TokenOut),
		AmountIn:     amountIn,
		MinAmountOut: minAmountOut,
		Deadline:     deadline,
		Nonce:        req.Nonce,
		Signature:    sig,
		ReceivedAt:   time.Now(),
	}

	// Non-blocking push: return 503 if the auction engine channel is saturated.
	select {
	case h.intentCh <- intent:
	default:
		http.Error(w, "auction engine at capacity, please retry", http.StatusServiceUnavailable)
		return
	}

	// Broadcast to solvers after handing off to the auction engine.
	h.hub.BroadcastIntent(intent)

	h.logger.Info("intent accepted",
		zap.String("intent_id", req.ID),
		zap.String("sender", req.Sender),
		zap.String("amount_in", req.AmountIn),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(intentResponse{
		IntentID: req.ID,
		Status:   "queued",
	}); err != nil {
		h.logger.Error("failed to write SubmitIntent response", zap.Error(err))
	}
}

// Health handles GET /health. Returns 200 OK for liveness probes.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		h.logger.Error("failed to write health response", zap.Error(err))
	}
}
