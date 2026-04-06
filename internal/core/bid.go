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

// Bid represents a solver's offer to fill a specific Intent.
// GasPrice is the solver's declared gas price, used for tie-breaking
// when two bids have equal AmountOut.
type Bid struct {
	IntentID   IntentID
	Solver     common.Address
	AmountOut  *big.Int
	GasPrice   *big.Int
	ReceivedAt time.Time
	Signature  []byte
}
