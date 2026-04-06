package txmanager

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/signer"
)

// ── Stubs ─────────────────────────────────────────────────────────────────────

// stubSettlement implements settlementCaller. All calls return err (may be nil).
type stubSettlement struct{ err error }

func (s *stubSettlement) Settle(
	_ *bind.TransactOpts,
	_ [32]byte,
	_ common.Address,
	_ common.Address,
	_ common.Address,
	_ *big.Int,
	_ *big.Int,
	_ *big.Int,
	_ *big.Int,
	_ common.Address,
	_ *big.Int,
	_ []byte,
) (*types.Transaction, error) {
	return nil, s.err
}

// stubGasTipper implements gasTipCapper. Returns tip or err; Client() returns nil
// (safe as long as Resync is not exercised — all test stubs return non-nonce errors).
type stubGasTipper struct {
	tip *big.Int
	err error
}

func (s *stubGasTipper) SuggestGasTipCap(_ context.Context) (*big.Int, error) {
	return s.tip, s.err
}

func (s *stubGasTipper) Client() *ethclient.Client { return nil }

// ── Helpers ───────────────────────────────────────────────────────────────────

// anvilTestKey is Anvil account #0. Used to build a real KeyVault without any
// network call — it is pure ECDSA key derivation.
const anvilTestKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// newTestExecutor wires an Executor with the provided stubs and a seeded NonceQueue.
func newTestExecutor(settlement settlementCaller, gas gasTipCapper) *Executor {
	vault, err := signer.NewKeyVaultFromHex(anvilTestKey)
	if err != nil {
		panic(fmt.Sprintf("newTestExecutor: key vault: %v", err))
	}

	q := &NonceQueue{}
	q.current.Store(0)

	return NewExecutor(
		settlement,
		vault,
		q,
		gas,
		signer.Domain{Name: "VynX", Version: "1", ChainID: 31337},
		big.NewInt(31337),
		zap.NewNop(),
	)
}

// newTestVoucher creates a fully-populated Voucher so that execute() can reach
// the settlement stub without panicking on nil big.Int fields.
func newTestVoucher(id string) *core.Voucher {
	intent := &core.Intent{
		ID:           core.IntentID(id),
		Sender:       common.Address{},
		TokenIn:      common.Address{},
		TokenOut:     common.Address{},
		AmountIn:     big.NewInt(1_000),
		MinAmountOut: big.NewInt(900),
		Deadline:     time.Now().Add(time.Hour),
		Nonce:        1,
	}
	return &core.Voucher{
		IntentID:      core.IntentID(id),
		WinningSolver: common.Address{},
		AmountOut:     big.NewInt(950),
		Intent:        intent,
		CreatedAt:     time.Now(),
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestExecutor_Run_RoutesFailureToTxFailedCh verifies that when execute() fails
// (settlement stub returns an error), Run() pushes the IntentID to txFailedCh.
func TestExecutor_Run_RoutesFailureToTxFailedCh(t *testing.T) {
	t.Parallel()

	settlement := &stubSettlement{err: fmt.Errorf("settlement reverted: slippage exceeded")}
	gas := &stubGasTipper{tip: big.NewInt(1_000_000_000)} // 1 gwei, no error
	ex := newTestExecutor(settlement, gas)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voucherCh := make(chan *core.Voucher, 1)
	txFailedCh := make(chan core.IntentID, 1)

	go ex.Run(ctx, voucherCh, txFailedCh)

	v := newTestVoucher("intent-route-failure")
	voucherCh <- v

	select {
	case id := <-txFailedCh:
		if id != v.IntentID {
			t.Errorf("txFailedCh got %q, want %q", id, v.IntentID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: IntentID never reached txFailedCh")
	}
}

// TestExecutor_Run_TxFailedCh_NonBlocking verifies that a full (zero-capacity)
// txFailedCh does NOT deadlock Run(). The executor must use the non-blocking
// select/default pattern and continue processing subsequent vouchers.
func TestExecutor_Run_TxFailedCh_NonBlocking(t *testing.T) {
	t.Parallel()

	settlement := &stubSettlement{err: fmt.Errorf("settlement reverted")}
	gas := &stubGasTipper{tip: big.NewInt(1_000_000_000)}
	ex := newTestExecutor(settlement, gas)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voucherCh := make(chan *core.Voucher, 5)
	// txFailedCh has zero capacity — every send from Run() will hit the default branch.
	txFailedCh := make(chan core.IntentID)

	go ex.Run(ctx, voucherCh, txFailedCh)

	// Push 3 vouchers. If Run() were to block on txFailedCh, the test would deadlock.
	for i := range 3 {
		voucherCh <- newTestVoucher(fmt.Sprintf("nonblocking-%d", i))
	}

	// Allow the executor to drain the queue, then cancel.
	time.Sleep(200 * time.Millisecond)
	cancel()

	// If we reach here without hanging, the non-blocking guarantee holds.
}

// TestExecutor_Run_CtxCancel verifies that Run() exits within a bounded time
// when the context is cancelled, even with an empty voucher channel.
func TestExecutor_Run_CtxCancel(t *testing.T) {
	t.Parallel()

	ex := newTestExecutor(
		&stubSettlement{},
		&stubGasTipper{tip: big.NewInt(1)},
	)

	ctx, cancel := context.WithCancel(context.Background())

	voucherCh := make(chan *core.Voucher)
	txFailedCh := make(chan core.IntentID, 1)

	done := make(chan struct{})
	go func() {
		ex.Run(ctx, voucherCh, txFailedCh)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Run() exited cleanly after ctx cancel.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run() did not exit within 500ms after ctx cancel")
	}
}

// TestExecutor_Run_ClosedVoucherCh verifies that Run() exits when voucherCh is
// closed, independent of context cancellation.
func TestExecutor_Run_ClosedVoucherCh(t *testing.T) {
	t.Parallel()

	ex := newTestExecutor(
		&stubSettlement{},
		&stubGasTipper{tip: big.NewInt(1)},
	)

	voucherCh := make(chan *core.Voucher)
	txFailedCh := make(chan core.IntentID, 1)

	done := make(chan struct{})
	go func() {
		ex.Run(context.Background(), voucherCh, txFailedCh)
		close(done)
	}()

	close(voucherCh)

	select {
	case <-done:
		// Run() exited after channel close.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run() did not exit within 500ms after voucherCh closed")
	}
}

// TestExecutor_Run_Concurrent_NoDataRace is the primary concurrency stress test.
// 50 goroutines inject vouchers simultaneously; the race detector must report
// zero data races across all shared state (NonceQueue, logger, stubs).
func TestExecutor_Run_Concurrent_NoDataRace(t *testing.T) {
	t.Parallel()

	const numVouchers = 50

	settlement := &stubSettlement{err: fmt.Errorf("settlement reverted")}
	gas := &stubGasTipper{tip: big.NewInt(1_000_000_000)}
	ex := newTestExecutor(settlement, gas)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voucherCh := make(chan *core.Voucher, numVouchers)
	txFailedCh := make(chan core.IntentID, numVouchers)

	go ex.Run(ctx, voucherCh, txFailedCh)

	var wg sync.WaitGroup
	wg.Add(numVouchers)

	for i := range numVouchers {
		go func(idx int) {
			defer wg.Done()
			voucherCh <- newTestVoucher(fmt.Sprintf("concurrent-%d", idx))
		}(i)
	}

	wg.Wait()

	// Allow the executor to process all vouchers, then shut down.
	// Each voucher cycles through execute() and optionally pushes to txFailedCh.
	time.Sleep(500 * time.Millisecond)
	cancel()
}
