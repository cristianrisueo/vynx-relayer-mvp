package auction

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// NewEngine allocates an auction Engine ready for use.
//
//   - logger: production Zap logger (use zap.NewNop() in tests).
//   - voucherBufSize: buffer depth of the internal voucher channel.
//     Set to at least the expected peak throughput to prevent auction goroutines
//     from blocking on a slow consumer.
//   - auctionTimeout: how long each auction waits for bids before closing.
//     The spec mandates 200 ms; tests may inject a shorter or longer value.
func NewEngine(logger *zap.Logger, voucherBufSize int, auctionTimeout time.Duration) *Engine {
	return &Engine{
		auctions:  make(map[core.IntentID]*auctionEntry),
		voucherCh: make(chan *core.Voucher, voucherBufSize),
		timeout:   auctionTimeout,
		logger:    logger,
	}
}

// Vouchers returns a receive-only channel on which completed Vouchers are published.
// The caller must drain this channel continuously; a full channel causes vouchers
// to be dropped (logged as ERROR).
func (e *Engine) Vouchers() <-chan *core.Voucher {
	return e.voucherCh
}

// StartAuction registers the intent and launches its auction goroutine.
// The goroutine lives for exactly e.timeout, then selects a winner and exits.
// Returns an error if an auction for this IntentID is already in flight.
func (e *Engine) StartAuction(intent *core.Intent) error {
	e.mu.Lock()
	if _, exists := e.auctions[intent.ID]; exists {
		e.mu.Unlock()
		return fmt.Errorf("failed to start auction for intent %s: auction already active", intent.ID)
	}
	entry := &auctionEntry{
		intent:    intent,
		bids:      make([]*core.Bid, 0, 64),
		startedAt: time.Now(),
	}
	e.auctions[intent.ID] = entry
	e.mu.Unlock()

	go e.runAuction(intent)
	return nil
}

// SubmitBid delivers a solver bid to the active auction for the bid's IntentID.
// Returns an error if no active auction exists (expired, cleaned up, or unknown),
// or if AmountOut / GasPrice are nil (which would panic in selectWinner).
func (e *Engine) SubmitBid(bid *core.Bid) error {
	if bid.AmountOut == nil {
		return fmt.Errorf("failed to submit bid for intent %s: AmountOut is nil", bid.IntentID)
	}
	if bid.GasPrice == nil {
		return fmt.Errorf("failed to submit bid for intent %s: GasPrice is nil", bid.IntentID)
	}
	if !e.submitBidLocked(bid) {
		return fmt.Errorf("failed to submit bid for intent %s: auction not found or expired", bid.IntentID)
	}
	return nil
}

// runAuction is the isolated goroutine governing a single auction lifecycle:
//  1. Wait for e.timeout (200 ms in production).
//  2. Select the winning bid from per-intent sharded state.
//  3. Emit a Voucher to the output channel (non-blocking; drops on full channel).
//  4. Call Cleanup to delete the entry from the in-memory map (mandatory GC).
func (e *Engine) runAuction(intent *core.Intent) {
	// Cleanup is always called — even when no winner is found — to prevent leaks.
	defer e.Cleanup(intent.ID)

	timer := time.NewTimer(e.timeout)
	defer timer.Stop()

	<-timer.C

	winner := e.selectWinner(intent.ID)
	if winner == nil {
		e.logger.Warn("auction closed with no bids",
			zap.String("intent_id", string(intent.ID)),
		)
		return
	}

	voucher := &core.Voucher{
		IntentID:      intent.ID,
		WinningSolver: winner.Solver,
		AmountOut:     winner.AmountOut,
		Intent:        intent,
		WinningBid:    winner,
		CreatedAt:     time.Now(),
	}

	// Non-blocking send: if the consumer is behind, drop and log rather than
	// stalling the auction goroutine (HFT mandate: no blocking I/O on hot path).
	select {
	case e.voucherCh <- voucher:
		e.logger.Info("voucher emitted",
			zap.String("intent_id", string(intent.ID)),
			zap.String("winner", winner.Solver.Hex()),
			zap.String("amount_out", winner.AmountOut.String()),
		)
	default:
		e.logger.Error("voucher channel full — dropping voucher",
			zap.String("intent_id", string(intent.ID)),
		)
	}
}
