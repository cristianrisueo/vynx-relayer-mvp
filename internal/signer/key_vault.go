// Package signer handles ECDSA key management and EIP-712 signing.
// The private key is loaded once at startup, stored only as *ecdsa.PrivateKey,
// and the raw hex bytes are zeroed immediately after parsing.
package signer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// KeyVault holds a single ECDSA private key for signing relayer transactions.
// The raw key material is never accessible after construction.
type KeyVault struct {
	priv *ecdsa.PrivateKey
	addr common.Address
}

// NewKeyVaultFromHex parses a 32-byte hex-encoded private key (no 0x prefix),
// zeroes the input byte slice immediately after use, and returns a KeyVault.
//
// Example env var: RELAYER_PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
func NewKeyVaultFromHex(hexKey string) (*KeyVault, error) {
	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key hex: %w", err)
	}

	priv, err := crypto.ToECDSA(b)
	// Zero the raw bytes regardless of success or failure.
	// clear() is the idiomatic Go 1.21+ builtin for zeroing a slice.
	clear(b)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ECDSA private key: %w", err)
	}

	return &KeyVault{
		priv: priv,
		addr: crypto.PubkeyToAddress(priv.PublicKey),
	}, nil
}

// Address returns the Ethereum address derived from the vault's public key.
func (v *KeyVault) Address() common.Address {
	return v.addr
}

// Sign produces a secp256k1 signature over a 32-byte digest in [R || S || V] format
// where V is 0 or 1. This is the raw ECDSA format expected by crypto.SigToPub.
// The caller is responsible for constructing a correct EIP-712 digest before calling.
func (v *KeyVault) Sign(digest [32]byte) ([]byte, error) {
	sig, err := crypto.Sign(digest[:], v.priv)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %w", err)
	}
	return sig, nil
}
