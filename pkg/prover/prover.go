package prover

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/circuit"
)

// Prover handles ZK proof generation using Groth16 on BN254.
type Prover struct {
	depth int

	cs constraint.ConstraintSystem
	pk groth16.ProvingKey
	vk groth16.VerifyingKey

	initialized bool
	mu          sync.RWMutex
}

// NewProver creates a new prover for the given tree depth.
// This performs the trusted setup (use MPC in production).
func NewProver(treeDepth int) (*Prover, error) {
	p := &Prover{
		depth: treeDepth,
	}

	if err := p.Setup(); err != nil {
		return nil, fmt.Errorf("prover setup failed: %w", err)
	}

	return p, nil
}

// Setup performs the circuit compilation and trusted setup.
// WARNING: This uses single-party setup. Use MPC for production.
func (p *Prover) Setup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create circuit template
	circuitDef := circuit.NewMembershipCircuit(p.depth)

	// Compile to R1CS
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuitDef)
	if err != nil {
		return fmt.Errorf("circuit compilation failed: %w", err)
	}
	p.cs = cs

	// Perform trusted setup
	pk, vk, err := groth16.Setup(cs)
	if err != nil {
		return fmt.Errorf("groth16 setup failed: %w", err)
	}

	p.pk = pk
	p.vk = vk
	p.initialized = true

	return nil
}

// Prove generates a Groth16 proof for the given witness.
func (p *Prover) Prove(witness *MembershipWitness) (*ProofResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("prover not initialized")
	}

	start := time.Now()

	// Create circuit assignment
	assignment := &circuit.MembershipCircuit{
		Secret:       witness.Secret,
		PathElements: make([]frontend.Variable, p.depth),
		PathIndices:  make([]frontend.Variable, p.depth),
		Root:         witness.Root,
		LeafHash:     witness.LeafHash,
	}

	// Fill in path elements and indices
	for i := 0; i < p.depth; i++ {
		if i < len(witness.PathElements) {
			assignment.PathElements[i] = witness.PathElements[i]
		} else {
			assignment.PathElements[i] = 0
		}
		if i < len(witness.PathIndices) {
			assignment.PathIndices[i] = witness.PathIndices[i]
		} else {
			assignment.PathIndices[i] = 0
		}
	}

	// Create witness
	fullWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	// Generate proof
	proof, err := groth16.Prove(p.cs, p.pk, fullWitness)
	if err != nil {
		return nil, fmt.Errorf("proof generation failed: %w", err)
	}

	// Serialize proof
	var proofBuf bytes.Buffer
	_, err = proof.WriteTo(&proofBuf)
	if err != nil {
		return nil, fmt.Errorf("proof serialization failed: %w", err)
	}

	duration := time.Since(start)

	return &ProofResult{
		Proof:      proof,
		ProofBytes: proofBuf.Bytes(),
		PublicInputs: &PublicInputs{
			Root:     witness.Root,
			LeafHash: witness.LeafHash,
		},
		Duration: duration,
	}, nil
}

// Verify verifies a proof against public inputs.
func (p *Prover) Verify(proofBytes []byte, publicInputs *PublicInputs) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return false, errors.New("prover not initialized")
	}

	// Deserialize proof
	proof := groth16.NewProof(ecc.BN254)
	_, err := proof.ReadFrom(bytes.NewReader(proofBytes))
	if err != nil {
		return false, fmt.Errorf("proof deserialization failed: %w", err)
	}

	// Create public witness
	publicAssignment := &circuit.MembershipCircuit{
		PathElements: make([]frontend.Variable, p.depth),
		PathIndices:  make([]frontend.Variable, p.depth),
		Root:         publicInputs.Root,
		LeafHash:     publicInputs.LeafHash,
	}

	publicWitness, err := frontend.NewWitness(publicAssignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create public witness: %w", err)
	}

	// Verify
	err = groth16.Verify(proof, p.vk, publicWitness)
	if err != nil {
		return false, nil // Proof is invalid, but not an error
	}

	return true, nil
}

// VerifyWithProof verifies using a groth16.Proof object directly.
func (p *Prover) VerifyWithProof(proof groth16.Proof, publicInputs *PublicInputs) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return false, errors.New("prover not initialized")
	}

	publicAssignment := &circuit.MembershipCircuit{
		PathElements: make([]frontend.Variable, p.depth),
		PathIndices:  make([]frontend.Variable, p.depth),
		Root:         publicInputs.Root,
		LeafHash:     publicInputs.LeafHash,
	}

	publicWitness, err := frontend.NewWitness(publicAssignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create public witness: %w", err)
	}

	err = groth16.Verify(proof, p.vk, publicWitness)
	return err == nil, nil
}

// GetVerificationKey returns the verification key.
func (p *Prover) GetVerificationKey() (groth16.VerifyingKey, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("prover not initialized")
	}

	return p.vk, nil
}

// GetProvingKey returns the proving key.
func (p *Prover) GetProvingKey() (groth16.ProvingKey, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("prover not initialized")
	}

	return p.pk, nil
}

// ExportVerificationKey serializes the verification key.
func (p *Prover) ExportVerificationKey() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("prover not initialized")
	}

	var buf bytes.Buffer
	_, err := p.vk.WriteTo(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Depth returns the tree depth this prover was configured for.
func (p *Prover) Depth() int {
	return p.depth
}

// ConstraintCount returns the number of constraints in the circuit.
func (p *Prover) ConstraintCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.cs == nil {
		return 0
	}
	return p.cs.GetNbConstraints()
}

// IsInitialized returns whether the prover has been set up.
func (p *Prover) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}
