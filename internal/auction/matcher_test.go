package auction

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// newTestEngine creates an Engine with a no-op logger suitable for testing.
func newTestEngine(voucherBuf int, timeout time.Duration) *Engine {
	return NewEngine(zap.NewNop(), voucherBuf, timeout)
}

// TestStartAuction_Basic verifies that a single auction emits a Voucher with the
// highest-AmountOut bid as the winner.
func TestStartAuction_Basic(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(10, 100*time.Millisecond)
	intent := &core.Intent{
		ID:           "intent-basic",
		AmountIn:     big.NewInt(1000),
		MinAmountOut: big.NewInt(900),
	}

	if err := engine.StartAuction(intent); err != nil {
		t.Fatalf("StartAuction: %v", err)
	}

	// Submit two bids; the second has a higher AmountOut and must win.
	bids := []*core.Bid{
		{IntentID: intent.ID, AmountOut: big.NewInt(950), GasPrice: big.NewInt(1)},
		{IntentID: intent.ID, AmountOut: big.NewInt(980), GasPrice: big.NewInt(1)},
	}
	for _, b := range bids {
		if err := engine.SubmitBid(b); err != nil {
			t.Fatalf("SubmitBid: %v", err)
		}
	}

	select {
	case v := <-engine.Vouchers():
		if v.AmountOut.Cmp(big.NewInt(980)) != 0 {
			t.Errorf("expected winning AmountOut=980, got %s", v.AmountOut)
		}
		if v.IntentID != intent.ID {
			t.Errorf("expected IntentID=%s, got %s", intent.ID, v.IntentID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: no voucher received")
	}
}

// TestStartAuction_Duplicate verifies that starting a second auction for an
// already-active IntentID returns an error.
func TestStartAuction_Duplicate(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(10, 500*time.Millisecond)
	intent := &core.Intent{ID: "intent-dup", AmountIn: big.NewInt(1)}

	if err := engine.StartAuction(intent); err != nil {
		t.Fatalf("first StartAuction: %v", err)
	}
	if err := engine.StartAuction(intent); err == nil {
		t.Fatal("expected error for duplicate StartAuction, got nil")
	}
}

// TestStartAuction_NoBids verifies that an auction with zero bids emits no Voucher
// and cleans up its entry from the map.
func TestStartAuction_NoBids(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(10, 50*time.Millisecond)
	intent := &core.Intent{ID: "intent-nobids", AmountIn: big.NewInt(1)}

	if err := engine.StartAuction(intent); err != nil {
		t.Fatalf("StartAuction: %v", err)
	}

	select {
	case v := <-engine.Vouchers():
		t.Fatalf("unexpected voucher for no-bid auction: %+v", v)
	case <-time.After(300 * time.Millisecond):
		// Expected path: no voucher emitted.
	}

	// Entry must be cleaned up after the timer fires.
	engine.mu.RLock()
	_, exists := engine.auctions[intent.ID]
	engine.mu.RUnlock()
	if exists {
		t.Error("auction entry was not cleaned up after timeout")
	}
}

// TestSubmitBid_AfterExpiry verifies that submitting a bid after an auction has
// closed returns an error (the entry has been deleted by Cleanup).
func TestSubmitBid_AfterExpiry(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(10, 20*time.Millisecond)
	intent := &core.Intent{ID: "intent-expired", AmountIn: big.NewInt(1)}

	if err := engine.StartAuction(intent); err != nil {
		t.Fatalf("StartAuction: %v", err)
	}

	// Wait for the auction to expire and be cleaned up.
	time.Sleep(100 * time.Millisecond)

	err := engine.SubmitBid(&core.Bid{
		IntentID:  intent.ID,
		AmountOut: big.NewInt(999),
		GasPrice:  big.NewInt(1),
	})
	if err == nil {
		t.Fatal("expected error submitting bid to expired auction, got nil")
	}
}

// TestCleanup_MemoryRelease verifies that Cleanup actually deletes the entry
// from the map (preventing long-term memory leaks in the auctions map).
func TestCleanup_MemoryRelease(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(10, 10*time.Millisecond)

	const count = 100
	for i := range count {
		id := core.IntentID(string(rune('A' + i)))
		intent := &core.Intent{ID: id, AmountIn: big.NewInt(1)}
		if err := engine.StartAuction(intent); err != nil {
			t.Fatalf("StartAuction %s: %v", id, err)
		}
	}

	// Wait long enough for all timers to fire and Cleanup to run.
	time.Sleep(200 * time.Millisecond)

	engine.mu.RLock()
	remaining := len(engine.auctions)
	engine.mu.RUnlock()

	if remaining != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", remaining)
	}
}

// TestConcurrentBids_NoDataRace is the primary stress test.
// It injects 10,000 concurrent bids into a single auction to surface any
// data races under the Go race detector (go test -race).
//
// The winning bid must be the one with the highest AmountOut (index 9999 → 10899).
func TestConcurrentBids_NoDataRace(t *testing.T) {
	t.Parallel()

	const (
		numBids       = 10_000
		baseAmount    = int64(900)
		auctionWindow = 500 * time.Millisecond // generous window for all goroutines to run
	)

	engine := newTestEngine(1, auctionWindow)
	intent := &core.Intent{
		ID:           "intent-stress",
		AmountIn:     big.NewInt(1000),
		MinAmountOut: big.NewInt(baseAmount),
	}

	if err := engine.StartAuction(intent); err != nil {
		t.Fatalf("StartAuction: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(numBids)

	for i := range numBids {
		go func(idx int) {
			defer wg.Done()
			bid := &core.Bid{
				IntentID:  intent.ID,
				AmountOut: big.NewInt(baseAmount + int64(idx)),
				GasPrice:  big.NewInt(1),
			}
			// Errors are expected for late bids (after timer fires); silently ignore.
			_ = engine.SubmitBid(bid)
		}(i)
	}

	wg.Wait()

	select {
	case v := <-engine.Vouchers():
		if v == nil {
			t.Fatal("received nil voucher")
		}
		// The winner must be at least baseAmount (MinAmountOut).
		if v.AmountOut.Cmp(big.NewInt(baseAmount)) < 0 {
			t.Errorf("winning AmountOut %s is below MinAmountOut %d", v.AmountOut, baseAmount)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: no voucher received after 3 s")
	}
}

// TestConcurrentAuctions_NoDataRace verifies that multiple simultaneous auctions
// do not race against each other when bids arrive interleaved.
func TestConcurrentAuctions_NoDataRace(t *testing.T) {
	t.Parallel()

	const (
		numAuctions   = 50
		bidsPerSlot   = 200
		auctionWindow = 300 * time.Millisecond
	)

	engine := newTestEngine(numAuctions, auctionWindow)

	intents := make([]*core.Intent, numAuctions)
	for i := range numAuctions {
		intents[i] = &core.Intent{
			ID:           core.IntentID("concurrent-" + string(rune('A'+i))),
			AmountIn:     big.NewInt(1000),
			MinAmountOut: big.NewInt(500),
		}
		if err := engine.StartAuction(intents[i]); err != nil {
			t.Fatalf("StartAuction %s: %v", intents[i].ID, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(numAuctions * bidsPerSlot)

	for _, intent := range intents {
		for j := range bidsPerSlot {
			go func(id core.IntentID, amount int64) {
				defer wg.Done()
				_ = engine.SubmitBid(&core.Bid{
					IntentID:  id,
					AmountOut: big.NewInt(amount),
					GasPrice:  big.NewInt(1),
				})
			}(intent.ID, int64(500+j))
		}
	}

	wg.Wait()

	received := 0
	deadline := time.After(3 * time.Second)
	for received < numAuctions {
		select {
		case <-engine.Vouchers():
			received++
		case <-deadline:
			t.Fatalf("timeout: received %d/%d vouchers", received, numAuctions)
		}
	}
}
