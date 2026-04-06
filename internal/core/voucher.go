// Package core defines the domain types shared across all VynX Relayer slices.
// No logic lives here — only plain data structures and named ID types.
// Keeping types in a leaf package with zero internal imports is the mechanism
// that allows the Event Bus pattern in main.go to wire slices without cycles.
package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Voucher is the settlement proof emitted by the auction engine after a winner
// is selected. It is passed to the TxManager for on-chain execution.
type Voucher struct {
	IntentID      IntentID
	WinningSolver common.Address
	AmountOut     *big.Int
	Intent        *Intent
	WinningBid    *Bid
	CreatedAt     time.Time
}
