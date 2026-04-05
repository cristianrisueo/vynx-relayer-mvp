// Package core defines the domain types shared across all VynX Relayer slices.
// No logic lives here — only plain data structures.
package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// IntentID is a unique identifier for a solver intent.
// Using a named string type prevents accidental mixing with other string IDs.
type IntentID string

// Intent represents a user's swap intent submitted to the VynX relayer.
// All monetary amounts use *big.Int to match on-chain precision (18 decimals).
// Deadline is a Go time.Time; callers must convert to Unix uint256 for on-chain use.
type Intent struct {
	ID           IntentID
	Sender       common.Address
	TokenIn      common.Address
	TokenOut     common.Address
	AmountIn     *big.Int
	MinAmountOut *big.Int
	Deadline     time.Time
	Nonce        uint64
	Signature    []byte
	ReceivedAt   time.Time
}
