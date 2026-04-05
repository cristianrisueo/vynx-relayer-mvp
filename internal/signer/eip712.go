package signer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
)

// EIP-712 type strings. These must match the Solidity struct definitions
// in VynxSettlement.sol exactly — any deviation breaks signature verification.
const (
	domainTypeString = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
	intentTypeString = "Intent(address sender,address tokenIn,address tokenOut,uint256 amountIn,uint256 minAmountOut,uint256 deadline,uint256 nonce)"
)

// Type hashes are computed once at package init. They are effectively immutable
// and safe for concurrent reads. [32]byte cannot be const in Go, so var is used.
var (
	domainTypeHash = crypto.Keccak256Hash([]byte(domainTypeString))
	intentTypeHash = crypto.Keccak256Hash([]byte(intentTypeString))
)

// Domain carries the EIP-712 domain parameters for the VynX protocol.
// These must match the values passed to the VynxSettlement constructor.
type Domain struct {
	// Name must equal "VynX".
	Name string
	// Version must equal "1".
	Version string
	// ChainID is the EVM chain ID (31337 for Anvil, 8453 for Base mainnet).
	ChainID uint64
	// VerifyingContract is the deployed address of VynxSettlement.
	VerifyingContract common.Address
}

// DomainSeparator computes the EIP-712 domain separator for the given domain.
// The result is deterministic for a given (name, version, chainID, verifyingContract) tuple.
func DomainSeparator(domain Domain) ([32]byte, error) {
	nameHash := crypto.Keccak256Hash([]byte(domain.Name))
	versionHash := crypto.Keccak256Hash([]byte(domain.Version))

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create bytes32 ABI type: %w", err)
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create uint256 ABI type: %w", err)
	}
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create address ABI type: %w", err)
	}

	args := abi.Arguments{
		abi.Argument{Type: bytes32Type}, // typeHash
		abi.Argument{Type: bytes32Type}, // nameHash
		abi.Argument{Type: bytes32Type}, // versionHash
		abi.Argument{Type: uint256Type}, // chainId
		abi.Argument{Type: addressType}, // verifyingContract
	}

	encoded, err := args.Pack(
		domainTypeHash,
		nameHash,
		versionHash,
		new(big.Int).SetUint64(domain.ChainID),
		domain.VerifyingContract,
	)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to ABI-encode domain separator: %w", err)
	}

	return crypto.Keccak256Hash(encoded), nil
}

// HashIntent computes the final EIP-712 digest for an Intent.
// The digest is suitable for signing with KeyVault.Sign or verifying with RecoverSigner.
func HashIntent(domain Domain, intent *core.Intent) ([32]byte, error) {
	domSep, err := DomainSeparator(domain)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to compute domain separator for intent %s: %w", intent.ID, err)
	}

	structHash, err := intentStructHash(intent)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to compute struct hash for intent %s: %w", intent.ID, err)
	}

	// EIP-712 final digest: keccak256("\x19\x01" || domainSeparator || structHash)
	digest := crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domSep[:],
		structHash[:],
	)
	return digest, nil
}

// intentStructHash ABI-encodes and hashes a single Intent struct per EIP-712.
func intentStructHash(intent *core.Intent) ([32]byte, error) {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create bytes32 ABI type: %w", err)
	}
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create address ABI type: %w", err)
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create uint256 ABI type: %w", err)
	}

	args := abi.Arguments{
		abi.Argument{Type: bytes32Type}, // typeHash
		abi.Argument{Type: addressType}, // sender
		abi.Argument{Type: addressType}, // tokenIn
		abi.Argument{Type: addressType}, // tokenOut
		abi.Argument{Type: uint256Type}, // amountIn
		abi.Argument{Type: uint256Type}, // minAmountOut
		abi.Argument{Type: uint256Type}, // deadline (Unix timestamp)
		abi.Argument{Type: uint256Type}, // nonce
	}

	encoded, err := args.Pack(
		intentTypeHash,
		intent.Sender,
		intent.TokenIn,
		intent.TokenOut,
		intent.AmountIn,
		intent.MinAmountOut,
		big.NewInt(intent.Deadline.Unix()),
		new(big.Int).SetUint64(intent.Nonce),
	)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to ABI-encode intent struct: %w", err)
	}

	return crypto.Keccak256Hash(encoded), nil
}

// RecoverSigner recovers the Ethereum address that signed the given 32-byte digest
// using the provided 65-byte [R || S || V] signature. This function is pure —
// it does not access any KeyVault state.
//
// The V byte must be 0 or 1. If the caller receives a signature with V in {27, 28}
// (e.g., from MetaMask eth_sign), subtract 27 from V before passing here.
func RecoverSigner(hash [32]byte, sig []byte) (common.Address, error) {
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("invalid signature length: got %d, want 65", len(sig))
	}

	pubKey, err := crypto.SigToPub(hash[:], sig)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover public key from signature: %w", err)
	}

	return crypto.PubkeyToAddress(*pubKey), nil
}
