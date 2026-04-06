// Command simulate is a live end-to-end integration driver for the VynX Relayer.
// It acts simultaneously as a User (submitting an Intent via HTTP POST) and a
// Solver (receiving the intent broadcast over WebSocket and replying with a bid).
//
// Both keys are generated ephemerally in memory using crypto.GenerateKey and are
// discarded when the process exits — no .env key material is ever read.
//
// Prerequisites:
//
//	anvil --chain-id 31337                     (terminal 1)
//	SETTLEMENT_CONTRACT_ADDRESS=0x... make build && ./bin/relayer  (terminal 2)
//	make simulate                              (terminal 3)
//
// Environment overrides (all optional — defaults target localhost Anvil):
//
//	RELAYER_HTTP               HTTP base URL of the running relayer
//	RELAYER_WS                 WebSocket base URL of the running relayer
//	SETTLEMENT_CONTRACT_ADDRESS Deployed VynxSettlement address
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/signer"
)

// ── Defaults ──────────────────────────────────────────────────────────────────

const (
	defaultRelayerHTTP    = "http://localhost:8080"
	defaultRelayerWS      = "ws://localhost:8080"
	defaultChainID        = uint64(31337)
	defaultSettlementAddr = "0x5FbDB2315678afecb367f032d93F642f64180aa3" // anvil deploy slot 0

	// Intent amounts (decimal wei strings, matching ingress.intentRequest format).
	constAmountIn     = "1000000000000000000" // 1 ETH
	constMinAmountOut = "900000000000000000"  // 0.9 ETH
	constBidAmountOut = "950000000000000000"  // 0.95 ETH — solver offer
	constBidGasPrice  = "1000000000"          // 1 gwei

	// How long to wait after bid submission for auction close + on-chain settlement.
	// Auction window is 200 ms; 5 s is a generous budget for the full cycle.
	settlementWait = 5 * time.Second

	// Overall deadline for the simulation; prevents hanging if the relayer is down.
	simulationTimeout = 30 * time.Second
)

// ── JSON wire types ───────────────────────────────────────────────────────────

// intentHTTPRequest mirrors internal/ingress.intentRequest.
type intentHTTPRequest struct {
	ID           string `json:"id"`
	Sender       string `json:"sender"`
	TokenIn      string `json:"token_in"`
	TokenOut     string `json:"token_out"`
	AmountIn     string `json:"amount_in"`
	MinAmountOut string `json:"min_amount_out"`
	Deadline     int64  `json:"deadline"`
	Nonce        uint64 `json:"nonce"`
	Signature    string `json:"signature"` // hex-encoded EIP-712 sig
}

// wsBroadcast is the envelope sent by the relayer to connected solvers.
type wsBroadcast struct {
	Type   string          `json:"type"`
	Intent json.RawMessage `json:"intent,omitempty"`
}

// wsIntentPayload is the subset of fields the solver needs from a new_intent broadcast.
type wsIntentPayload struct {
	ID string `json:"id"`
}

// wsBidRequest mirrors internal/ingress.bidRequest — the JSON a solver sends to bid.
type wsBidRequest struct {
	IntentID  string `json:"intent_id"`
	Solver    string `json:"solver"`
	AmountOut string `json:"amount_out"`
	GasPrice  string `json:"gas_price"`
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	relayerHTTP := envOrDefault("RELAYER_HTTP", defaultRelayerHTTP)
	relayerWS := envOrDefault("RELAYER_WS", defaultRelayerWS)
	settlementAddr := common.HexToAddress(envOrDefault("SETTLEMENT_CONTRACT_ADDRESS", defaultSettlementAddr))

	// ── 1. Generate ephemeral keys ─────────────────────────────────────────────
	// crypto.GenerateKey uses crypto/rand internally; keys live only for the
	// duration of this process and are never written to disk.
	userKey, err := crypto.GenerateKey()
	must(err, "generate user key")
	solverKey, err := crypto.GenerateKey()
	must(err, "generate solver key")

	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	solverAddr := crypto.PubkeyToAddress(solverKey.PublicKey)

	printBanner(userAddr, solverAddr, settlementAddr, relayerHTTP)

	// ── 2. Global context ──────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), simulationTimeout)
	defer cancel()

	// ── 3. Connect WebSocket (register as solver) ──────────────────────────────
	logf("[1] Connecting WebSocket solver channel → %s/v1/ws/solvers", relayerWS)
	wsConn, _, err := websocket.DefaultDialer.DialContext( //nolint:gosec
		ctx, relayerWS+"/v1/ws/solvers", nil,
	)
	if err != nil {
		fatalf("WebSocket dial failed: %v\n\n    Ensure the relayer is running: make build && ./bin/relayer", err)
	}
	defer func() { _ = wsConn.Close() }()
	logf("    Connected.")

	// ── 4. Build Intent ────────────────────────────────────────────────────────
	intentID := fmt.Sprintf("sim-%d", time.Now().UnixNano())
	deadline := time.Now().Add(5 * time.Minute)
	const nonce = uint64(1)

	// Mainnet USDC / WETH addresses are used as dummies; no actual ERC-20
	// interaction occurs — the relayer only needs structurally valid addresses.
	tokenIn := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	tokenOut := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")

	amtIn, ok := new(big.Int).SetString(constAmountIn, 10)
	if !ok {
		fatalf("invalid constAmountIn constant")
	}
	minAmt, ok := new(big.Int).SetString(constMinAmountOut, 10)
	if !ok {
		fatalf("invalid constMinAmountOut constant")
	}

	intent := &core.Intent{
		ID:           core.IntentID(intentID),
		Sender:       userAddr,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		AmountIn:     amtIn,
		MinAmountOut: minAmt,
		Deadline:     deadline,
		Nonce:        nonce,
	}

	logf("[2] Intent built: id=%s", intentID)

	// ── 5. EIP-712 sign ────────────────────────────────────────────────────────
	logf("[3] Signing Intent with EIP-712 ...")

	domain := signer.Domain{
		Name:              "VynX",
		Version:           "1",
		ChainID:           defaultChainID,
		VerifyingContract: settlementAddr,
	}

	digest, err := signer.HashIntent(domain, intent)
	must(err, "compute EIP-712 digest")

	// Sign directly with the ephemeral key — no need for KeyVault since this key
	// is already in memory and we don't need the zeroing guarantee for a test key.
	sig, err := crypto.Sign(digest[:], userKey)
	must(err, "sign EIP-712 digest")

	sigHex := "0x" + hex.EncodeToString(sig)
	logf("    Digest:    0x%x", digest)
	logf("    Signature: %s...%s", sigHex[:10], sigHex[len(sigHex)-8:])

	// ── 6. WS goroutine — listen for broadcast, reply with bid ────────────────
	bidSent := make(chan struct{})
	go func() {
		defer close(bidSent)
		for {
			_, raw, readErr := wsConn.ReadMessage()
			if readErr != nil {
				// Connection was closed by main or timeout; exit silently.
				return
			}

			var envelope wsBroadcast
			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				logf("[WS] unparseable message: %v", jsonErr)
				continue
			}
			if envelope.Type != "new_intent" {
				continue
			}

			var payload wsIntentPayload
			if jsonErr := json.Unmarshal(envelope.Intent, &payload); jsonErr != nil {
				logf("[WS] unparseable intent payload: %v", jsonErr)
				continue
			}
			logf("[5] Intent broadcast received (id: %s)", payload.ID)

			bid := wsBidRequest{
				IntentID:  payload.ID,
				Solver:    solverAddr.Hex(),
				AmountOut: constBidAmountOut,
				GasPrice:  constBidGasPrice,
			}
			bidBytes, marshalErr := json.Marshal(bid)
			if marshalErr != nil {
				logf("[WS] bid marshal error: %v", marshalErr)
				return
			}
			if writeErr := wsConn.WriteMessage(websocket.TextMessage, bidBytes); writeErr != nil {
				logf("[WS] bid send error: %v", writeErr)
				return
			}
			logf("[6] Bid submitted: solver=%s amount_out=%s gas_price=%s",
				solverAddr.Hex()[:10]+"...", constBidAmountOut, constBidGasPrice)
			return // one bid per simulation run
		}
	}()

	// ── 7. POST Intent to relayer ──────────────────────────────────────────────
	httpBody := intentHTTPRequest{
		ID:           intentID,
		Sender:       userAddr.Hex(),
		TokenIn:      tokenIn.Hex(),
		TokenOut:     tokenOut.Hex(),
		AmountIn:     constAmountIn,
		MinAmountOut: constMinAmountOut,
		Deadline:     deadline.Unix(),
		Nonce:        nonce,
		Signature:    sigHex,
	}

	bodyBytes, err := json.Marshal(httpBody)
	must(err, "marshal intent HTTP request")

	logf("[4] POST %s/v1/intent ...", relayerHTTP)

	postReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		relayerHTTP+"/v1/intent", //nolint:gosec
		bytes.NewReader(bodyBytes),
	)
	must(err, "build HTTP request")
	postReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(postReq)
	if err != nil {
		fatalf("POST /v1/intent failed: %v\n\n    Is the relayer running?", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var respBody map[string]string
	if jsonErr := json.NewDecoder(resp.Body).Decode(&respBody); jsonErr != nil {
		fatalf("decode POST response: %v", jsonErr)
	}
	logf("    → %d %s  (intent_id: %s, status: %s)",
		resp.StatusCode,
		http.StatusText(resp.StatusCode),
		respBody["intent_id"],
		respBody["status"],
	)

	if resp.StatusCode != http.StatusAccepted {
		fatalf("relayer rejected intent (status %d) — check relayer logs", resp.StatusCode)
	}

	// ── 8. Wait for bid confirmation, then allow settlement cycle to complete ──
	select {
	case <-bidSent:
		// Bid goroutine sent the bid and exited.
	case <-time.After(10 * time.Second):
		fatalf("timeout waiting for intent broadcast — check relayer logs")
	}

	logf("[7] Waiting %s for auction close + on-chain settlement ...", settlementWait)
	time.Sleep(settlementWait)

	logf("[8] Simulation complete.")
	logf("    Check relayer logs for 'settlement submitted' or 'failed to settle'.")
	logf("    (Settlement requires a deployed contract; expected log on Anvil: 'settlement submitted')")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// must exits the process with a formatted fatal message if err is non-nil.
// Intentional use of os.Exit here; this is a CLI driver, not library code,
// so panic or structured error propagation would be overly ceremonial.
func must(err error, context string) {
	if err != nil {
		fatalf("%s: %v", context, err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFATAL: "+format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func printBanner(user, solver, settlement common.Address, relayerHTTP string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║          VynX Live Simulation (Anvil)            ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Printf("  User (ephemeral):    %s\n", user.Hex())
	fmt.Printf("  Solver (ephemeral):  %s\n", solver.Hex())
	fmt.Printf("  Settlement contract: %s\n", settlement.Hex())
	fmt.Printf("  Relayer:             %s\n", relayerHTTP)
	fmt.Println()
}
