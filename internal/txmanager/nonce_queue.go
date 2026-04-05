// Package txmanager handles the L2 execution slice: nonce management, OP Stack gas
// estimation, and on-chain settlement via the VynxSettlement contract.
package txmanager

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// NonceQueue manages the relayer's transaction nonce entirely in RAM using a
// lock-free atomic counter. This guarantees sequential nonce assignment at
// sub-microsecond latency without any global mutex on the hot path.
//
// Resync Design: when a transaction is rejected by the node with a nonce error
// (e.g. "nonce too low"), the caller must invoke Resync to re-anchor the counter
// against the node's authoritative pending nonce. Resync does a single RPC call
// and atomically stores the result — subsequent Next() calls resume from there.
type NonceQueue struct {
	current atomic.Uint64
}

// NewNonceQueue seeds the in-RAM counter from the node's current pending nonce
// for addr. Returns an error if the RPC call fails — the caller must not proceed
// without a valid starting nonce.
func NewNonceQueue(ctx context.Context, client *ethclient.Client, addr common.Address) (*NonceQueue, error) {
	n, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to seed nonce queue for address %s: %w", addr.Hex(), err)
	}
	q := &NonceQueue{}
	q.current.Store(n)
	return q, nil
}

// Next atomically reserves and returns the next nonce value for a transaction.
// The internal counter is incremented before the caller receives the value, so
// concurrent calls from multiple goroutines always produce distinct nonces.
//
// The call is wait-free: it completes in O(1) regardless of contention.
func (q *NonceQueue) Next() uint64 {
	// Add(1) returns the post-increment value; subtract 1 to obtain the reserved nonce.
	return q.current.Add(1) - 1
}

// Resync discards the in-RAM counter and re-fetches the authoritative pending
// nonce from the RPC node. It must be called whenever a transaction is rejected
// with a nonce-desync error to prevent the relayer from emitting duplicate or
// out-of-order nonces in recovery mode.
//
// IMPORTANT: Resync should only be called when all in-flight transactions that
// were issued with the old nonce range have either been confirmed or dropped from
// the mempool. Calling Resync while a transaction with nonce N is still pending
// can cause the counter to roll back to N, producing a duplicate nonce if another
// goroutine calls Next() concurrently. In the current single-executor design this
// is safe — execute() is called sequentially — but callers must account for this
// if parallelism is ever added.
//
// Resync is safe for concurrent use: the underlying store is atomic, so a
// concurrent Next() call will either observe the old value (still valid if it
// races before the store) or the re-synced value (correct for the next batch).
func (q *NonceQueue) Resync(ctx context.Context, client *ethclient.Client, addr common.Address) error {
	n, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		return fmt.Errorf("failed to resync nonce for address %s: %w", addr.Hex(), err)
	}
	q.current.Store(n)
	return nil
}
