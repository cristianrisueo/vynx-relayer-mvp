// Package main tests the Event Bus routing wired in main.go without starting
// the HTTP server or connecting to a real RPC node.
//
// These are white-box tests (package main) so they can call the unexported
// dispatcher helpers directly — no test-only exports needed.
package main

import (
	"context"
	"math/big"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/auction"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// TestEventBus_IntentToVoucher is the primary E2E routing test.
// It injects one Intent and one Bid through the Event Bus channels and verifies
// that a Voucher with the correct winner emerges from the auction engine within
// the 250 ms SLA mandated by the spec.
//
// No RPC, no executor, no HTTP — only in-memory channels and the auction engine.
//
// Pipeline:
//
//	intentCh → runIntentDispatcher → auction.Engine.StartAuction
//	bidCh    → runBidDispatcher    → auction.Engine.SubmitBid
//	                                  → voucherCh  ← assert here
func TestEventBus_IntentToVoucher(t *testing.T) {
	t.Parallel()

	const (
		auctionWindow = 50 * time.Millisecond  // short window so the test completes fast
		e2eDeadline   = 250 * time.Millisecond // SLA budget
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zap.NewNop()

	intentCh := make(chan *core.Intent, 10)
	bidCh := make(chan *core.Bid, 10)

	engine := auction.NewEngine(logger, 10, auctionWindow)

	go runIntentDispatcher(ctx, intentCh, engine, logger)
	go runBidDispatcher(ctx, bidCh, engine, logger)

	intent := &core.Intent{
		ID:           "e2e-intent-1",
		AmountIn:     big.NewInt(1_000),
		MinAmountOut: big.NewInt(900),
		Deadline:     time.Now().Add(time.Hour),
	}
	intentCh <- intent

	// Allow the intent dispatcher to call StartAuction before the bid arrives.
	// A 10 ms pause is orders of magnitude above a goroutine context switch (~1 µs).
	time.Sleep(10 * time.Millisecond)

	bid := &core.Bid{
		IntentID:  intent.ID,
		AmountOut: big.NewInt(950),
		GasPrice:  big.NewInt(1),
	}
	bidCh <- bid

	select {
	case v := <-engine.Vouchers():
		if v.IntentID != intent.ID {
			t.Errorf("voucher IntentID = %q, want %q", v.IntentID, intent.ID)
		}
		if v.AmountOut.Cmp(big.NewInt(950)) != 0 {
			t.Errorf("voucher AmountOut = %s, want 950", v.AmountOut)
		}
	case <-time.After(e2eDeadline):
		t.Fatalf("no voucher received within %s SLA", e2eDeadline)
	}
}

// TestEventBus_MultiBid_WinnerIsHighest verifies that when multiple bids arrive
// for the same intent the auction engine selects the highest AmountOut.
func TestEventBus_MultiBid_WinnerIsHighest(t *testing.T) {
	t.Parallel()

	const auctionWindow = 60 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zap.NewNop()

	intentCh := make(chan *core.Intent, 1)
	bidCh := make(chan *core.Bid, 10)

	engine := auction.NewEngine(logger, 10, auctionWindow)

	go runIntentDispatcher(ctx, intentCh, engine, logger)
	go runBidDispatcher(ctx, bidCh, engine, logger)

	intent := &core.Intent{
		ID:           "e2e-multibid",
		AmountIn:     big.NewInt(1_000),
		MinAmountOut: big.NewInt(900),
		Deadline:     time.Now().Add(time.Hour),
	}
	intentCh <- intent

	time.Sleep(10 * time.Millisecond) // let StartAuction run

	// Three competing bids; 980 must win.
	bids := []int64{920, 980, 950}
	for _, amount := range bids {
		bidCh <- &core.Bid{
			IntentID:  intent.ID,
			AmountOut: big.NewInt(amount),
			GasPrice:  big.NewInt(1),
		}
	}

	select {
	case v := <-engine.Vouchers():
		if v.AmountOut.Cmp(big.NewInt(980)) != 0 {
			t.Errorf("winner AmountOut = %s, want 980", v.AmountOut)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("no voucher received within 250ms")
	}
}

// TestEventBus_CtxCancel_DispatchersExit verifies that both dispatcher goroutines
// exit promptly when the context is cancelled, preventing goroutine leaks on
// graceful shutdown.
func TestEventBus_CtxCancel_DispatchersExit(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	logger := zap.NewNop()
	engine := auction.NewEngine(logger, 10, 100*time.Millisecond)

	intentCh := make(chan *core.Intent, 10)
	bidCh := make(chan *core.Bid, 10)

	intentDone := make(chan struct{})
	bidDone := make(chan struct{})

	go func() {
		runIntentDispatcher(ctx, intentCh, engine, logger)
		close(intentDone)
	}()
	go func() {
		runBidDispatcher(ctx, bidCh, engine, logger)
		close(bidDone)
	}()

	cancel()

	deadline := time.After(500 * time.Millisecond)

	for intentDone != nil || bidDone != nil {
		select {
		case <-intentDone:
			intentDone = nil
		case <-bidDone:
			bidDone = nil
		case <-deadline:
			t.Fatal("dispatcher goroutine(s) did not exit within 500ms of ctx cancel")
		}
	}
}
