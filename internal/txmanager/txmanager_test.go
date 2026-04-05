package txmanager

import (
	"sync"
	"testing"
)

// TestNonceQueue_Sequential verifies that consecutive Next() calls return strictly
// sequential nonces with no gaps, starting from the seeded value.
func TestNonceQueue_Sequential(t *testing.T) {
	t.Parallel()

	const (
		seed  = uint64(42)
		count = 1_000
	)

	q := &NonceQueue{}
	q.current.Store(seed)

	for i := range count {
		got := q.Next()
		want := seed + uint64(i)
		if got != want {
			t.Fatalf("Next()[%d] = %d, want %d", i, got, want)
		}
	}
}

// TestNonceQueue_Concurrent_NoDataRace is the primary stress test for the nonce queue.
// It fires 10,000 concurrent goroutines each calling Next() once and verifies that:
//
//  1. No nonce is issued twice (zero duplicates).
//  2. The full range [0, 10000) is covered with no gaps.
//
// Run with go test -race to surface any data races in the atomic implementation.
func TestNonceQueue_Concurrent_NoDataRace(t *testing.T) {
	t.Parallel()

	const numGoroutines = 10_000

	q := &NonceQueue{}
	q.current.Store(0)

	nonces := make([]uint64, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			nonces[idx] = q.Next() // each goroutine writes to its own index — no data race
		}(i)
	}

	wg.Wait()

	// Verify uniqueness: build a frequency map and flag any duplicates.
	seen := make(map[uint64]struct{}, numGoroutines)
	for _, n := range nonces {
		if _, exists := seen[n]; exists {
			t.Fatalf("duplicate nonce detected: %d", n)
		}
		seen[n] = struct{}{}
	}

	// Verify completeness: every value in [0, numGoroutines) must be present.
	for i := range numGoroutines {
		if _, exists := seen[uint64(i)]; !exists {
			t.Fatalf("missing nonce: %d (gap in atomic counter)", i)
		}
	}
}

// TestNonceQueue_Resync verifies that after Resync the counter is re-anchored to
// the provided value and Next() resumes from that point.
// This test exercises Resync without a live RPC node by calling Store directly —
// the same operation Resync performs after PendingNonceAt succeeds.
func TestNonceQueue_Resync(t *testing.T) {
	t.Parallel()

	q := &NonceQueue{}
	q.current.Store(5)

	// Simulate three successful transactions.
	for range 3 {
		q.Next()
	}
	if got := q.Next(); got != 8 {
		t.Fatalf("expected nonce 8 after 3 increments from 5, got %d", got)
	}

	// Simulate a Resync that re-anchors to nonce 3 (node-reported pending nonce).
	q.current.Store(3)

	if got := q.Next(); got != 3 {
		t.Fatalf("expected nonce 3 after resync, got %d", got)
	}
}

// TestIsNonceError_Patterns verifies that all known nonce-error strings are correctly
// identified, and that unrelated errors are not false-positives.
func TestIsNonceError_Patterns(t *testing.T) {
	t.Parallel()

	type testCase struct {
		msg  string
		want bool
	}

	cases := []testCase{
		{"nonce too low", true},
		{"Nonce Too Low", true}, // case-insensitive
		{"replacement transaction underpriced", true},
		{"already known", true},
		{"out of gas", false},
		{"insufficient funds", false},
		{"execution reverted", false},
		{"", false},
	}

	for _, tc := range cases {
		err := &mockError{tc.msg}
		if got := isNonceError(err); got != tc.want {
			t.Errorf("isNonceError(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

// mockError is a minimal error implementation for table-driven tests.
type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }
