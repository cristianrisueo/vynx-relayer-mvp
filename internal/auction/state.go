// Package auction implements the in-memory Order Flow Auction (OFA) engine.
// Each auction is an isolated goroutine; state is sharded per-intent to avoid
// any global hot-path locks.
package auction

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// auctionEntry holds all bids received for a single in-flight intent.
// The mutex is scoped to this entry only (Lock Sharding): concurrent bid
// submissions for *different* intents never contend on the same lock.
type auctionEntry struct {
	mu        sync.Mutex
	bids      []*core.Bid
	intent    *core.Intent
	startedAt time.Time
}

// Engine is the in-memory auction registry.
//
// Locking hierarchy (must always be acquired in this order to prevent deadlocks):
//
//  1. Engine.mu  – guards the auctions map (add / remove entries)
//  2. auctionEntry.mu – guards the bids slice within a single entry
//
// The outer lock is held for the shortest possible duration: only to look up
// or register an entry pointer, never during bid appending or winner selection.
type Engine struct {
	mu        sync.RWMutex
	auctions  map[core.IntentID]*auctionEntry
	voucherCh chan *core.Voucher
	timeout   time.Duration
	logger    *zap.Logger
}

// submitBidLocked appends the bid to the entry's slice under the entry's own mutex.
// Returns false if no active auction exists for the bid's intent ID.
func (e *Engine) submitBidLocked(bid *core.Bid) bool {
	e.mu.RLock()
	entry := e.auctions[bid.IntentID]
	e.mu.RUnlock()

	// No active auction (expired or unknown intent). Caller decides how to handle.
	if entry == nil {
		return false
	}

	entry.mu.Lock()
	entry.bids = append(entry.bids, bid)
	entry.mu.Unlock()
	return true
}

// selectWinner picks the bid with the highest AmountOut for the given intent.
// Ties are broken by the highest GasPrice.
// Returns nil if the entry does not exist or no bids were received.
func (e *Engine) selectWinner(id core.IntentID) *core.Bid {
	// Read-lock the map only to obtain the entry pointer, then release immediately.
	e.mu.RLock()
	entry := e.auctions[id]
	e.mu.RUnlock()

	if entry == nil {
		return nil
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	var winner *core.Bid
	for _, bid := range entry.bids {
		if winner == nil {
			winner = bid
			continue
		}
		cmp := bid.AmountOut.Cmp(winner.AmountOut)
		if cmp > 0 || (cmp == 0 && bid.GasPrice.Cmp(winner.GasPrice) > 0) {
			winner = bid
		}
	}
	return winner
}

// Cleanup removes the auction entry for the given intent from the in-memory map.
// It MUST be called exactly once after the auction goroutine completes.
// Failing to call Cleanup causes a permanent memory leak in the auctions map.
func (e *Engine) Cleanup(id core.IntentID) {
	e.mu.Lock()
	delete(e.auctions, id)
	e.mu.Unlock()
}
