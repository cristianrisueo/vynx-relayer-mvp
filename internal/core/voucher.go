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
