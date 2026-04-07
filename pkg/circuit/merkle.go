package circuit

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// MerkleVerifier provides Merkle proof verification within circuits.
type MerkleVerifier struct {
	api    frontend.API
	hasher mimc.MiMC
}

// NewMerkleVerifier creates a new Merkle verifier for use in circuits.
func NewMerkleVerifier(api frontend.API) (*MerkleVerifier, error) {
	hasher, err := mimc.NewMiMC(api)
	if err != nil {
		return nil, err
	}
	return &MerkleVerifier{api: api, hasher: hasher}, nil
}

// VerifyProof verifies a Merkle proof within a circuit.
// Returns the computed root for comparison.
func (v *MerkleVerifier) VerifyProof(
	leaf frontend.Variable,
	pathElements []frontend.Variable,
	pathIndices []frontend.Variable,
) frontend.Variable {
	currentHash := leaf

	for i := 0; i < len(pathElements); i++ {
		sibling := pathElements[i]
		direction := pathIndices[i]

		// Ensure direction is binary
		v.api.AssertIsBoolean(direction)

		// Select left and right based on direction
		left := v.api.Select(direction, sibling, currentHash)
		right := v.api.Select(direction, currentHash, sibling)

		// Compute parent hash
		v.hasher.Reset()
		v.hasher.Write(left)
		v.hasher.Write(right)
		currentHash = v.hasher.Sum()
	}

	return currentHash
}

// HashLeaf computes the hash of a leaf value.
func (v *MerkleVerifier) HashLeaf(value frontend.Variable) frontend.Variable {
	v.hasher.Reset()
	v.hasher.Write(value)
	return v.hasher.Sum()
}

// HashPair computes the hash of two values.
func (v *MerkleVerifier) HashPair(left, right frontend.Variable) frontend.Variable {
	v.hasher.Reset()
	v.hasher.Write(left)
	v.hasher.Write(right)
	return v.hasher.Sum()
}

// ComputeMerkleRoot computes the Merkle root from a leaf and proof.
// This is a convenience wrapper around VerifyProof.
func ComputeMerkleRoot(
	api frontend.API,
	leaf frontend.Variable,
	pathElements []frontend.Variable,
	pathIndices []frontend.Variable,
) (frontend.Variable, error) {
	verifier, err := NewMerkleVerifier(api)
	if err != nil {
		return nil, err
	}
	return verifier.VerifyProof(leaf, pathElements, pathIndices), nil
}
