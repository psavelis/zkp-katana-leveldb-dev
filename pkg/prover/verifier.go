package prover

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/circuit"
)

// Verifier provides standalone proof verification using a verification key.
type Verifier struct {
	vk    groth16.VerifyingKey
	depth int
}

// NewVerifier creates a verifier from a serialized verification key.
func NewVerifier(vkBytes []byte, depth int) (*Verifier, error) {
	vk := groth16.NewVerifyingKey(ecc.BN254)
	_, err := vk.ReadFrom(bytes.NewReader(vkBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize verification key: %w", err)
	}

	return &Verifier{
		vk:    vk,
		depth: depth,
	}, nil
}

// NewVerifierFromKey creates a verifier from a verification key object.
func NewVerifierFromKey(vk groth16.VerifyingKey, depth int) *Verifier {
	return &Verifier{
		vk:    vk,
		depth: depth,
	}
}

// Verify verifies a proof against public inputs.
func (v *Verifier) Verify(proofBytes []byte, publicInputs *PublicInputs) (bool, error) {
	if v.vk == nil {
		return false, errors.New("verifier not initialized")
	}

	// Deserialize proof
	proof := groth16.NewProof(ecc.BN254)
	_, err := proof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return false, fmt.Errorf("proof deserialization failed: %w", err)
	}

	// Create public witness
	publicAssignment := &circuit.MembershipCircuit{
		PathElements: make([]frontend.Variable, v.depth),
		PathIndices:  make([]frontend.Variable, v.depth),
		Root:         publicInputs.Root,
		LeafHash:     publicInputs.LeafHash,
	}

	publicWitness, err := frontend.NewWitness(publicAssignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create public witness: %w", err)
	}

	// Verify
	err = groth16.Verify(proof, v.vk, publicWitness)
	if err != nil {
		return false, nil // Invalid proof, not an error
	}

	return true, nil
}

// VerifyWithProofData verifies using ProofData struct.
func (v *Verifier) VerifyWithProofData(proofData *ProofData) (bool, error) {
	publicInputs, err := NewPublicInputs(proofData.Root, proofData.LeafHash)
	if err != nil {
		return false, fmt.Errorf("failed to parse public inputs: %w", err)
	}

	return v.Verify(proofData.Proof, publicInputs)
}

// ExportVerificationKey returns the serialized verification key.
func (v *Verifier) ExportVerificationKey() ([]byte, error) {
	if v.vk == nil {
		return nil, errors.New("verifier not initialized")
	}

	var buf bytes.Buffer
	_, err := v.vk.WriteTo(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Depth returns the tree depth this verifier was configured for.
func (v *Verifier) Depth() int {
	return v.depth
}

// VerifyProofStandalone is a convenience function for one-off verification.
func VerifyProofStandalone(vkBytes, proofBytes []byte, root, leafHash []byte, depth int) (bool, error) {
	verifier, err := NewVerifier(vkBytes, depth)
	if err != nil {
		return false, err
	}

	publicInputs := NewPublicInputsFromBigInt(
		new(big.Int).SetBytes(root),
		new(big.Int).SetBytes(leafHash),
	)

	return verifier.Verify(proofBytes, publicInputs)
}
