// Command relayer is the VynX Core Relayer process.
// It wires all internal slices together via the Event Bus channel pattern,
// ensuring zero circular imports between packages.
//
// Event Bus topology (all channels created here, in main):
//
//	intentCh  (chan *core.Intent)   ingress.Handler → intent dispatcher → auction.Engine
//	bidCh     (chan *core.Bid)      ingress.Hub     → bid dispatcher    → auction.Engine
//	voucherCh (<-chan *core.Voucher) auction.Engine  →                   txmanager.Executor
//	txFailedCh(chan core.IntentID)  txmanager.Executor → failure drainer (MVP: log only)
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/cristianrisueo/vynx-relayer-mvp/bindings"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/auction"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/ingress"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/signer"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/txmanager"
)

func main() {
	// ── 1. Logger (must come first — everything else logs through it) ──────────
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, err := cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build zap logger: %v", err))
	}
	defer func() { _ = logger.Sync() }()

	// ── 2. Environment ─────────────────────────────────────────────────────────
	if loadErr := loadDotEnv(".env"); loadErr != nil {
		// .env is optional; env vars can be set by the shell instead.
		logger.Warn("could not load .env file", zap.Error(loadErr))
	}

	rpcURL := mustEnv(logger, "BASE_RPC_URL")
	privateKeyHex := mustEnv(logger, "RELAYER_PRIVATE_KEY")
	settlementAddrStr := mustEnv(logger, "SETTLEMENT_CONTRACT_ADDRESS")

	chainIDStr := mustEnv(logger, "CHAIN_ID")
	chainIDUint, parseErr := strconv.ParseUint(chainIDStr, 10, 64)
	if parseErr != nil {
		logger.Fatal("invalid CHAIN_ID", zap.String("raw", chainIDStr), zap.Error(parseErr))
	}

	auctionMSStr := os.Getenv("AUCTION_TIMEOUT_MS")
	if auctionMSStr == "" {
		auctionMSStr = "200"
	}
	auctionMS, parseErr := strconv.ParseUint(auctionMSStr, 10, 64)
	if parseErr != nil {
		logger.Fatal("invalid AUCTION_TIMEOUT_MS", zap.String("raw", auctionMSStr), zap.Error(parseErr))
	}
	auctionTimeout := time.Duration(auctionMS) * time.Millisecond

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// ── 3. ECDSA Key Vault ─────────────────────────────────────────────────────
	vault, err := signer.NewKeyVaultFromHex(privateKeyHex)
	if err != nil {
		logger.Fatal("failed to initialise key vault", zap.Error(err))
	}
	logger.Info("key vault initialised", zap.String("address", vault.Address().Hex()))

	// ── 4. Ethereum RPC client ─────────────────────────────────────────────────
	ethCl, err := ethclient.Dial(rpcURL)
	if err != nil {
		logger.Fatal("failed to connect to Base RPC", zap.String("url", rpcURL), zap.Error(err))
	}
	defer ethCl.Close()
	logger.Info("connected to Base RPC", zap.String("url", rpcURL))

	// ── 5. OP Stack client + nonce queue ───────────────────────────────────────
	opClient, err := txmanager.NewOPStackClient(ethCl)
	if err != nil {
		logger.Fatal("failed to build OP Stack client", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nonceQueue, err := txmanager.NewNonceQueue(ctx, ethCl, vault.Address())
	if err != nil {
		logger.Fatal("failed to seed nonce queue", zap.Error(err))
	}
	logger.Info("nonce queue seeded", zap.String("relayer", vault.Address().Hex()))

	// ── 6. Settlement contract binding ────────────────────────────────────────
	settlementAddr := common.HexToAddress(settlementAddrStr)
	settlementContract, err := bindings.NewVynxSettlement(settlementAddr, ethCl)
	if err != nil {
		logger.Fatal("failed to bind VynxSettlement contract", zap.Error(err))
	}

	// ── 7. Event Bus channels ──────────────────────────────────────────────────
	const (
		intentBuf = 256
		bidBuf    = 10_000
		txFailBuf = 128
	)
	intentCh := make(chan *core.Intent, intentBuf)
	bidCh := make(chan *core.Bid, bidBuf)
	txFailedCh := make(chan core.IntentID, txFailBuf)

	// ── 8. Auction engine ──────────────────────────────────────────────────────
	const voucherBuf = 256
	auctionEngine := auction.NewEngine(logger, voucherBuf, auctionTimeout)
	voucherCh := auctionEngine.Vouchers()

	// ── 9. TxManager executor ──────────────────────────────────────────────────
	executor := txmanager.NewExecutor(
		settlementContract,
		vault,
		nonceQueue,
		opClient,
		new(big.Int).SetUint64(chainIDUint),
		logger,
	)

	// ── 10. Ingress layer ──────────────────────────────────────────────────────
	hub := ingress.NewHub(bidCh, logger)
	handler := ingress.NewHandler(intentCh, hub, logger)

	// ── 11. Start background goroutines ───────────────────────────────────────
	go hub.Run(ctx)
	go runIntentDispatcher(ctx, intentCh, auctionEngine, logger)
	go runBidDispatcher(ctx, bidCh, auctionEngine, logger)
	go executor.Run(ctx, voucherCh, txFailedCh)
	go drainTxFailed(ctx, txFailedCh, logger)

	// ── 12. HTTP server ────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/v1/intent", handler.SubmitIntent)
	mux.HandleFunc("/v1/ws/solvers", hub.ServeWS)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", srv.Addr))
		if srvErr := srv.ListenAndServe(); !errors.Is(srvErr, http.ErrServerClosed) {
			logger.Error("HTTP server error", zap.Error(srvErr))
			cancel()
		}
	}()

	// ── 13. Graceful shutdown ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received — initiating graceful shutdown")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
		logger.Error("HTTP server shutdown timed out", zap.Error(shutdownErr))
	}
	logger.Info("relayer stopped")
}

// ── Event Bus dispatchers ──────────────────────────────────────────────────────

// runIntentDispatcher reads Intents from intentCh and starts an auction for each.
// It is the bridge between the ingress layer and the auction engine, keeping both
// packages free of direct imports of each other.
func runIntentDispatcher(
	ctx context.Context,
	ch <-chan *core.Intent,
	engine *auction.Engine,
	logger *zap.Logger,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case intent, ok := <-ch:
			if !ok {
				return
			}
			if err := engine.StartAuction(intent); err != nil {
				logger.Error("failed to start auction",
					zap.String("intent_id", string(intent.ID)),
					zap.Error(err),
				)
			}
		}
	}
}

// runBidDispatcher reads Bids from bidCh and submits each to the active auction.
func runBidDispatcher(
	ctx context.Context,
	ch <-chan *core.Bid,
	engine *auction.Engine,
	logger *zap.Logger,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case bid, ok := <-ch:
			if !ok {
				return
			}
			if err := engine.SubmitBid(bid); err != nil {
				// Expected on late bids (after auction timer expires); log at Debug.
				logger.Debug("bid rejected",
					zap.String("intent_id", string(bid.IntentID)),
					zap.Error(err),
				)
			}
		}
	}
}

// drainTxFailed consumes the txFailedCh Event Bus channel. In the MVP this logs
// failures only; a future slice can re-queue intents or penalise solvers here
// without touching txmanager or auction packages.
func drainTxFailed(ctx context.Context, ch <-chan core.IntentID, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-ch:
			if !ok {
				return
			}
			logger.Warn("on-chain settlement failed — no retry in MVP",
				zap.String("intent_id", string(id)),
			)
		}
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────────

// mustEnv returns the value of the environment variable key or calls logger.Fatal.
func mustEnv(logger *zap.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		logger.Fatal("required environment variable not set", zap.String("key", key))
	}
	return v
}

// loadDotEnv reads key=value pairs from path and sets them via os.Setenv.
// Lines beginning with # and blank lines are ignored. Existing env vars are
// NOT overridden — shell-level exports always take precedence.
func loadDotEnv(path string) error {
	f, err := os.Open(path) //nolint:gosec // path is a hardcoded constant, not user-controlled input
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Do not overwrite variables already present in the environment.
		if os.Getenv(key) == "" {
			if setErr := os.Setenv(key, value); setErr != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, setErr)
			}
		}
	}
	return scanner.Err()
}
