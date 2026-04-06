// Package prover provides Groth16 proof generation and verification services.
package prover

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"

	"github.com/consensys/gnark/backend/groth16"
)

// ProofData represents a serializable proof with metadata.
type ProofData struct {
	// Proof is the serialized Groth16 proof bytes.
	Proof []byte `json:"proof"`
	// PublicInputs contains the public inputs as hex strings.
	PublicInputs []string `json:"public_inputs"`
	// Root is the Merkle root (hex encoded).
	Root string `json:"root"`
	// LeafHash is the leaf hash / nullifier (hex encoded).
	LeafHash string `json:"leaf_hash"`
	// CreatedAt is the timestamp when the proof was generated.
	CreatedAt time.Time `json:"created_at"`
	// TreeDepth is the depth of the Merkle tree.
	TreeDepth int `json:"tree_depth"`
	// ProofTime is how long proof generation took.
	ProofTimeMs int64 `json:"proof_time_ms"`
}

// PublicInputs represents the public inputs for verification.
type PublicInputs struct {
	Root     *big.Int
	LeafHash *big.Int
}

// NewPublicInputs creates public inputs from hex strings.
func NewPublicInputs(rootHex, leafHashHex string) (*PublicInputs, error) {
	rootBytes, err := hex.DecodeString(rootHex)
	if err != nil {
		return nil, err
	}
	leafBytes, err := hex.DecodeString(leafHashHex)
	if err != nil {
		return nil, err
	}

	return &PublicInputs{
		Root:     new(big.Int).SetBytes(rootBytes),
		LeafHash: new(big.Int).SetBytes(leafBytes),
	}, nil
}

// NewPublicInputsFromBigInt creates public inputs from big.Int values.
func NewPublicInputsFromBigInt(root, leafHash *big.Int) *PublicInputs {
	return &PublicInputs{
		Root:     new(big.Int).Set(root),
		LeafHash: new(big.Int).Set(leafHash),
	}
}

// MembershipWitness contains the full witness data for proof generation.
type MembershipWitness struct {
	// Secret is the secret preimage (private).
	Secret []byte
	// PathElements are the Merkle proof siblings (private).
	PathElements []*big.Int
	// PathIndices indicate the path direction at each level (private).
	PathIndices []int
	// Root is the Merkle root (public).
	Root *big.Int
	// LeafHash is the hash of the secret (public).
	LeafHash *big.Int
}

// VerificationKey represents a serializable verification key.
type VerificationKey struct {
	// Data is the serialized verification key.
	Data []byte `json:"data"`
	// TreeDepth is the depth this key was generated for.
	TreeDepth int `json:"tree_depth"`
	// CreatedAt is when the key was generated.
	CreatedAt time.Time `json:"created_at"`
}

// ProvingKey represents a serializable proving key.
type ProvingKey struct {
	// Data is the serialized proving key.
	Data []byte `json:"data"`
	// TreeDepth is the depth this key was generated for.
	TreeDepth int `json:"tree_depth"`
	// CreatedAt is when the key was generated.
	CreatedAt time.Time `json:"created_at"`
}

// ProofResult is returned from the Prove method.
type ProofResult struct {
	// Proof is the gnark proof object.
	Proof groth16.Proof
	// ProofBytes is the serialized proof.
	ProofBytes []byte
	// PublicInputs are the public inputs used.
	PublicInputs *PublicInputs
	// Duration is how long proof generation took.
	Duration time.Duration
}

// ToProofData converts a ProofResult to storable ProofData.
func (r *ProofResult) ToProofData() *ProofData {
	return &ProofData{
		Proof: r.ProofBytes,
		PublicInputs: []string{
			hex.EncodeToString(r.PublicInputs.Root.Bytes()),
			hex.EncodeToString(r.PublicInputs.LeafHash.Bytes()),
		},
		Root:        hex.EncodeToString(r.PublicInputs.Root.Bytes()),
		LeafHash:    hex.EncodeToString(r.PublicInputs.LeafHash.Bytes()),
		CreatedAt:   time.Now(),
		ProofTimeMs: r.Duration.Milliseconds(),
	}
}

// ToJSON serializes ProofData to JSON.
func (p *ProofData) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// ProofDataFromJSON deserializes ProofData from JSON.
func ProofDataFromJSON(data []byte) (*ProofData, error) {
	var pd ProofData
	err := json.Unmarshal(data, &pd)
	return &pd, err
}

// SetupResult contains the results of circuit setup.
type SetupResult struct {
	ProvingKey      groth16.ProvingKey
	VerificationKey groth16.VerifyingKey
	Duration        time.Duration
}
