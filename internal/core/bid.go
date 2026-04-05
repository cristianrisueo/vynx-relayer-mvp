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
