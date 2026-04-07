// Package circuit provides gnark circuit definitions for ZK proofs.
package circuit

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// MembershipCircuit proves knowledge of a secret that hashes to a leaf
// in a Merkle tree with a known root.
//
// Public inputs:
//   - Root: The Merkle tree root
//   - LeafHash: The hash of the secret (can be used as nullifier)
//
// Private inputs (witness):
//   - Secret: The secret preimage
//   - PathElements: Sibling hashes along the Merkle path
//   - PathIndices: Direction indicators (0 = left, 1 = right)
type MembershipCircuit struct {
	// Private inputs
	Secret       frontend.Variable   `gnark:"secret"`
	PathElements []frontend.Variable `gnark:"pathElements,secret"`
	PathIndices  []frontend.Variable `gnark:"pathIndices,secret"`

	// Public inputs
	Root     frontend.Variable `gnark:",public"`
	LeafHash frontend.Variable `gnark:",public"`
}

// Define implements the circuit logic for gnark.
func (c *MembershipCircuit) Define(api frontend.API) error {
	// 1. Hash the secret to get the leaf
	hasher, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	hasher.Write(c.Secret)
	computedLeaf := hasher.Sum()

	// 2. Verify that the computed leaf matches the public leaf hash
	api.AssertIsEqual(computedLeaf, c.LeafHash)

	// 3. Verify the Merkle proof
	currentHash := computedLeaf
	for i := 0; i < len(c.PathElements); i++ {
		// Get the sibling and direction
		sibling := c.PathElements[i]
		direction := c.PathIndices[i]

		// Ensure direction is binary (0 or 1)
		api.AssertIsBoolean(direction)

		// Compute the parent hash based on direction
		// If direction == 0: current is left child, hash(current, sibling)
		// If direction == 1: current is right child, hash(sibling, current)
		left := api.Select(direction, sibling, currentHash)
		right := api.Select(direction, currentHash, sibling)

		hasher.Reset()
		hasher.Write(left)
		hasher.Write(right)
		currentHash = hasher.Sum()
	}

	// 4. Verify computed root matches the public root
	api.AssertIsEqual(currentHash, c.Root)

	return nil
}

// NewMembershipCircuit creates a new membership circuit with the specified depth.
func NewMembershipCircuit(depth int) *MembershipCircuit {
	return &MembershipCircuit{
		PathElements: make([]frontend.Variable, depth),
		PathIndices:  make([]frontend.Variable, depth),
	}
}

// GetDepth returns the tree depth of the circuit.
func (c *MembershipCircuit) GetDepth() int {
	return len(c.PathElements)
}
