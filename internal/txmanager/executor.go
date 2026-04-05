package txmanager

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/cristianrisueo/vynx-relayer-mvp/bindings"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/signer"
)

// Executor is the sole consumer of the auction voucher channel and the sole writer
// to the VynxSettlement contract on Base L2. It bridges the off-chain OFA engine
// with the on-chain settlement layer.
//
// Dependency Injection: Executor receives voucherCh and txFailedCh as plain
// channel arguments in Run(). This keeps internal/txmanager free of any import
// of internal/auction — communication is via channels wired in cmd/relayer/main.go
// (the Event Bus pattern mandated by the spec).
type Executor struct {
	settlement *bindings.VynxSettlement
	vault      *signer.KeyVault
	nonceQueue *NonceQueue
	opClient   *OPStackClient
	eipDomain  signer.Domain
	chainID    *big.Int
	logger     *zap.Logger
}

// NewExecutor constructs an Executor with all L2 execution dependencies injected.
func NewExecutor(
	settlement *bindings.VynxSettlement,
	vault *signer.KeyVault,
	nonceQueue *NonceQueue,
	opClient *OPStackClient,
	eipDomain signer.Domain,
	chainID *big.Int,
	logger *zap.Logger,
) *Executor {
	return &Executor{
		settlement: settlement,
		vault:      vault,
		nonceQueue: nonceQueue,
		opClient:   opClient,
		eipDomain:  eipDomain,
		chainID:    chainID,
		logger:     logger,
	}
}

// Run is the Executor's main event loop. It blocks until ctx is cancelled or
// voucherCh is closed, consuming each Voucher and dispatching a settle() call
// to the L2 contract.
//
// On failure, the IntentID is pushed to txFailedCh (non-blocking) so the
// auction engine can apply penalties or trigger a re-auction without importing
// this package — maintaining zero circular dependencies.
func (e *Executor) Run(ctx context.Context, voucherCh <-chan *core.Voucher, txFailedCh chan<- core.IntentID) {
	for {
		select {
		case <-ctx.Done():
			e.logger.Info("executor shutting down", zap.String("reason", ctx.Err().Error()))
			return
		case v, ok := <-voucherCh:
			if !ok {
				e.logger.Info("voucher channel closed; executor exiting")
				return
			}
			if err := e.execute(ctx, v); err != nil {
				e.logger.Error("failed to execute settlement",
					zap.String("intent_id", string(v.IntentID)),
					zap.Error(err),
				)
				// Non-blocking send: do not stall the executor if the failure
				// channel consumer is behind (HFT hot-path mandate).
				select {
				case txFailedCh <- v.IntentID:
				default:
					e.logger.Warn("txFailedCh full — dropping failed intent notification",
						zap.String("intent_id", string(v.IntentID)),
					)
				}
			}
		}
	}
}

// execute signs the EIP-712 intent digest and submits the settle() transaction
// to the L2 contract for a single Voucher. On nonce-desync errors it resyncs
// the NonceQueue so the next execution attempt uses the authoritative nonce.
func (e *Executor) execute(ctx context.Context, v *core.Voucher) error {
	// ── Step 1: EIP-712 sign ───────────────────────────────────────────────
	digest, err := signer.HashIntent(e.eipDomain, v.Intent)
	if err != nil {
		return fmt.Errorf("failed to hash intent %s for signing: %w", v.IntentID, err)
	}
	sig, err := e.vault.Sign(digest)
	if err != nil {
		return fmt.Errorf("failed to sign digest for intent %s: %w", v.IntentID, err)
	}

	// ── Step 2: Encode intentID as bytes32 ────────────────────────────────
	var intentIDBytes [32]byte
	copy(intentIDBytes[:], []byte(v.Intent.ID))

	// ── Step 3: Fetch gas tip cap from OP Stack node ───────────────────────
	tip, err := e.opClient.SuggestGasTipCap(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas tip cap for intent %s: %w", v.IntentID, err)
	}

	// ── Step 4: Reserve atomic nonce ──────────────────────────────────────
	nonce := e.nonceQueue.Next()

	// ── Step 5: Build TransactOpts with KeyVault signer closure ───────────
	// KeyVault.Sign is used inside a bind.SignerFn to keep the raw private key
	// confined to key_vault.go throughout the transaction signing lifecycle.
	vaultAddr := e.vault.Address()
	chainID := e.chainID

	auth := &bind.TransactOpts{
		From:      vaultAddr,
		Nonce:     new(big.Int).SetUint64(nonce),
		GasTipCap: tip,
		// GasFeeCap: nil → bind package fetches baseFee and computes 2×baseFee+tip.
		// GasLimit:  0   → bind package calls EstimateGas automatically.
		Context: ctx,
		Signer: func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if addr != vaultAddr {
				return nil, fmt.Errorf("signer address mismatch: got %s, want %s",
					addr.Hex(), vaultAddr.Hex())
			}
			ethSigner := types.LatestSignerForChainID(chainID)
			hash := ethSigner.Hash(tx)
			txSig, signErr := e.vault.Sign(hash)
			if signErr != nil {
				return nil, fmt.Errorf("failed to sign transaction for intent %s: %w",
					v.IntentID, signErr)
			}
			return tx.WithSignature(ethSigner, txSig)
		},
	}

	// ── Step 6: Call settle() on the VynxSettlement contract ──────────────
	_, txErr := e.settlement.Settle(
		auth,
		intentIDBytes,
		v.Intent.Sender,
		v.Intent.TokenIn,
		v.Intent.TokenOut,
		v.Intent.AmountIn,
		v.Intent.MinAmountOut,
		big.NewInt(v.Intent.Deadline.Unix()),
		new(big.Int).SetUint64(v.Intent.Nonce),
		v.WinningSolver,
		v.AmountOut,
		sig,
	)
	if txErr != nil {
		if isNonceError(txErr) {
			// Nonce desync: re-anchor the counter from the node before returning.
			// The next execute() call will use the authoritative pending nonce.
			e.logger.Warn("nonce desync detected — resyncing NonceQueue",
				zap.String("intent_id", string(v.IntentID)),
				zap.Uint64("skipped_nonce", nonce),
				zap.Error(txErr),
			)
			if resyncErr := e.nonceQueue.Resync(ctx, e.opClient.Client(), vaultAddr); resyncErr != nil {
				e.logger.Error("failed to resync nonce queue",
					zap.String("intent_id", string(v.IntentID)),
					zap.Error(resyncErr),
				)
			}
		} else {
			// Non-nonce error (gas, revert, etc.): the reserved nonce is now a gap
			// in the sequence. Log at Warn so on-call can diagnose mempool stalls.
			e.logger.Warn("settlement failed — nonce gap introduced",
				zap.String("intent_id", string(v.IntentID)),
				zap.Uint64("skipped_nonce", nonce),
				zap.Error(txErr),
			)
		}
		return fmt.Errorf("failed to settle intent %s: %w", v.IntentID, txErr)
	}

	e.logger.Info("settlement submitted",
		zap.String("intent_id", string(v.IntentID)),
		zap.String("winning_solver", v.WinningSolver.Hex()),
		zap.String("amount_out", v.AmountOut.String()),
		zap.Uint64("nonce", nonce),
	)
	return nil
}

// isNonceError returns true for any error indicating the transaction was rejected
// due to a stale or duplicate nonce. These patterns cover geth, Base, and
// standard OP Stack sequencer responses.
func isNonceError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nonce too low") ||
		strings.Contains(msg, "replacement transaction underpriced") ||
		strings.Contains(msg, "already known")
}
