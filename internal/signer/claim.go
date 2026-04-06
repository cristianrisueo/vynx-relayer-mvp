// Package signer — claim.go implements the EIP-191 claim signature scheme
// used by VynxSettlement.claimFunds().
//
// The signed payload matches the Solidity expression:
//
//	MessageHashUtils.toEthSignedMessageHash(keccak256(abi.encodePacked(intentId, solver)))
//
// This binds both the escrow identity and the recipient address, preventing
// front-running substitution of the solver field.
package signer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ClaimDigest computes the EIP-191 digest that the VynxSettlement contract
// expects in claimFunds(). The result can be signed directly by KeyVault.Sign.
//
// Solidity equivalent:
//
//	bytes32 messageHash = keccak256(abi.encodePacked(intentId, solver));
//	bytes32 digest = MessageHashUtils.toEthSignedMessageHash(messageHash);
func ClaimDigest(intentID [32]byte, solver common.Address) [32]byte {
	// keccak256(abi.encodePacked(intentId, solver))
	// intentId is 32 bytes; solver is 20 bytes = 52 bytes total.
	packed := make([]byte, 0, 52)
	packed = append(packed, intentID[:]...)
	packed = append(packed, solver.Bytes()...)
	messageHash := crypto.Keccak256Hash(packed)

	// toEthSignedMessageHash: "\x19Ethereum Signed Message:\n32" + messageHash
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	prefixed := make([]byte, 0, len(prefix)+32)
	prefixed = append(prefixed, prefix...)
	prefixed = append(prefixed, messageHash.Bytes()...)
	return crypto.Keccak256Hash(prefixed)
}
