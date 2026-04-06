// Package katana provides Starknet RPC client integration for Katana.
package katana

import (
	"encoding/hex"
	"math/big"
)

// Config holds Katana client configuration.
type Config struct {
	// RPCURL is the Katana RPC endpoint (default: http://localhost:5050)
	RPCURL string
	// ChainID is the chain identifier
	ChainID string
	// VerifierAddress is the deployed verifier contract address
	VerifierAddress string
	// PrivateKey for signing transactions (hex encoded, without 0x prefix)
	PrivateKey string
	// AccountAddress is the sender account address
	AccountAddress string
}

// DefaultConfig returns default configuration for local Katana.
func DefaultConfig() *Config {
	return &Config{
		RPCURL:  "http://localhost:5050",
		ChainID: "KATANA",
	}
}

// SimulationResult contains the result of a contract simulation.
type SimulationResult struct {
	Valid       bool   `json:"valid"`
	GasUsed     uint64 `json:"gas_used"`
	BlockNumber uint64 `json:"block_number"`
	Error       string `json:"error,omitempty"`
}

// TransactionResult contains the result of a submitted transaction.
type TransactionResult struct {
	TransactionHash string `json:"transaction_hash"`
	BlockNumber     uint64 `json:"block_number"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}

// BlockInfo contains block information.
type BlockInfo struct {
	BlockNumber uint64 `json:"block_number"`
	BlockHash   string `json:"block_hash"`
	Timestamp   uint64 `json:"timestamp"`
	ParentHash  string `json:"parent_hash"`
}

// ContractInfo contains deployed contract information.
type ContractInfo struct {
	Address   string `json:"address"`
	ClassHash string `json:"class_hash"`
}

// G1Point represents a point on BN254 G1 curve.
type G1Point struct {
	X *big.Int `json:"x"`
	Y *big.Int `json:"y"`
}

// G2Point represents a point on BN254 G2 curve.
type G2Point struct {
	X0 *big.Int `json:"x0"`
	X1 *big.Int `json:"x1"`
	Y0 *big.Int `json:"y0"`
	Y1 *big.Int `json:"y1"`
}

// Groth16Proof represents a Groth16 proof for on-chain verification.
type Groth16Proof struct {
	A G1Point `json:"a"`
	B G2Point `json:"b"`
	C G1Point `json:"c"`
}

// VerificationRequest represents a verification request to the contract.
type VerificationRequest struct {
	Proof        Groth16Proof `json:"proof"`
	MerkleRoot   *big.Int     `json:"merkle_root"`
	LeafHash     *big.Int     `json:"leaf_hash"`
	PublicInputs []*big.Int   `json:"public_inputs"`
}

// ToFeltArray converts big.Int values to felt252 hex strings for Starknet.
func ToFeltArray(values []*big.Int) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = "0x" + hex.EncodeToString(v.Bytes())
	}
	return result
}

// ToFelt converts a big.Int to a felt252 hex string.
func ToFelt(v *big.Int) string {
	if v == nil {
		return "0x0"
	}
	return "0x" + hex.EncodeToString(v.Bytes())
}

// FromFelt converts a felt252 hex string to big.Int.
func FromFelt(s string) (*big.Int, error) {
	// Remove 0x prefix if present
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}
	// Handle empty or "0" case
	if s == "" || s == "0" {
		return big.NewInt(0), nil
	}
	// Pad odd-length hex strings
	if len(s)%2 != 0 {
		s = "0" + s
	}
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(bytes), nil
}

// ChainStatus represents the current chain status.
type ChainStatus struct {
	ChainID     string `json:"chain_id"`
	BlockNumber uint64 `json:"block_number"`
	BlockHash   string `json:"block_hash"`
	Syncing     bool   `json:"syncing"`
}
