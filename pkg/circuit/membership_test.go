package circuit

import (
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/merkle"
)

func TestMembershipCircuit_Compile(t *testing.T) {
	// Test that the circuit compiles successfully
	circuit := NewMembershipCircuit(10)

	_, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)
}

func TestMembershipCircuit_ValidProof(t *testing.T) {
	depth := 10
	tree := merkle.NewMerkleTree(depth)

	// Insert some leaves
	secrets := []string{"alice_secret", "bob_secret", "charlie_secret"}
	for _, s := range secrets {
		tree.InsertSecret([]byte(s))
	}

	// Get proof for alice
	aliceSecret := []byte("alice_secret")
	aliceLeafHash := merkle.HashToBigInt(aliceSecret)
	aliceProof, err := tree.GetProof(0)
	require.NoError(t, err)

	root := tree.GetRoot()

	// Create witness assignment
	circuit := NewMembershipCircuit(depth)

	assignment := &MembershipCircuit{
		Secret:       aliceSecret,
		PathElements: make([]frontend.Variable, depth),
		PathIndices:  make([]frontend.Variable, depth),
		Root:         root,
		LeafHash:     aliceLeafHash,
	}

	for i := 0; i < depth; i++ {
		assignment.PathElements[i] = aliceProof.PathElements[i]
		assignment.PathIndices[i] = aliceProof.PathIndices[i]
	}

	// Test the circuit
	err = test.IsSolved(circuit, assignment, ecc.BN254.ScalarField())
	require.NoError(t, err)
}

func TestMembershipCircuit_InvalidSecret(t *testing.T) {
	depth := 10
	tree := merkle.NewMerkleTree(depth)

	// Insert alice
	tree.InsertSecret([]byte("alice_secret"))

	// Get proof for alice but use wrong secret
	aliceProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	// Use wrong secret but correct proof path
	wrongSecret := []byte("wrong_secret")
	wrongLeafHash := merkle.HashToBigInt(wrongSecret)

	circuit := NewMembershipCircuit(depth)
	assignment := &MembershipCircuit{
		Secret:       wrongSecret,
		PathElements: make([]frontend.Variable, depth),
		PathIndices:  make([]frontend.Variable, depth),
		Root:         root,
		LeafHash:     wrongLeafHash, // Matching wrong leaf hash
	}

	for i := 0; i < depth; i++ {
		assignment.PathElements[i] = aliceProof.PathElements[i]
		assignment.PathIndices[i] = aliceProof.PathIndices[i]
	}

	// This should fail because the proof path doesn't match the wrong leaf
	err := test.IsSolved(circuit, assignment, ecc.BN254.ScalarField())
	assert.Error(t, err, "circuit should not be satisfied with wrong secret")
}

func TestMembershipCircuit_WrongRoot(t *testing.T) {
	depth := 10
	tree := merkle.NewMerkleTree(depth)

	secret := []byte("test_secret")
	tree.InsertSecret(secret)

	proof, _ := tree.GetProof(0)
	leafHash := merkle.HashToBigInt(secret)

	// Use wrong root
	wrongRoot := big.NewInt(999999)

	circuit := NewMembershipCircuit(depth)
	assignment := &MembershipCircuit{
		Secret:       secret,
		PathElements: make([]frontend.Variable, depth),
		PathIndices:  make([]frontend.Variable, depth),
		Root:         wrongRoot,
		LeafHash:     leafHash,
	}

	for i := 0; i < depth; i++ {
		assignment.PathElements[i] = proof.PathElements[i]
		assignment.PathIndices[i] = proof.PathIndices[i]
	}

	err := test.IsSolved(circuit, assignment, ecc.BN254.ScalarField())
	assert.Error(t, err, "circuit should not be satisfied with wrong root")
}

func TestMembershipCircuit_LeafHashMismatch(t *testing.T) {
	depth := 10
	tree := merkle.NewMerkleTree(depth)

	secret := []byte("test_secret")
	tree.InsertSecret(secret)

	proof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	// Use mismatched leaf hash
	wrongLeafHash := big.NewInt(12345)

	circuit := NewMembershipCircuit(depth)
	assignment := &MembershipCircuit{
		Secret:       secret,
		PathElements: make([]frontend.Variable, depth),
		PathIndices:  make([]frontend.Variable, depth),
		Root:         root,
		LeafHash:     wrongLeafHash, // Doesn't match hash(secret)
	}

	for i := 0; i < depth; i++ {
		assignment.PathElements[i] = proof.PathElements[i]
		assignment.PathIndices[i] = proof.PathIndices[i]
	}

	err := test.IsSolved(circuit, assignment, ecc.BN254.ScalarField())
	assert.Error(t, err, "circuit should not be satisfied with mismatched leaf hash")
}

func TestMembershipCircuit_MultipleLeaves(t *testing.T) {
	depth := 10
	tree := merkle.NewMerkleTree(depth)

	// Insert multiple members
	members := []string{"alice", "bob", "charlie", "david", "eve"}
	for _, m := range members {
		tree.InsertSecret([]byte(m))
	}

	root := tree.GetRoot()

	// Test proof for each member
	for i, member := range members {
		secret := []byte(member)
		leafHash := merkle.HashToBigInt(secret)
		proof, err := tree.GetProof(i)
		require.NoError(t, err)

		circuit := NewMembershipCircuit(depth)
		assignment := &MembershipCircuit{
			Secret:       secret,
			PathElements: make([]frontend.Variable, depth),
			PathIndices:  make([]frontend.Variable, depth),
			Root:         root,
			LeafHash:     leafHash,
		}

		for j := 0; j < depth; j++ {
			assignment.PathElements[j] = proof.PathElements[j]
			assignment.PathIndices[j] = proof.PathIndices[j]
		}

		err = test.IsSolved(circuit, assignment, ecc.BN254.ScalarField())
		require.NoError(t, err, "proof for member %s should be valid", member)
	}
}

func TestMembershipCircuit_Groth16(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Groth16 test in short mode")
	}

	depth := 10
	tree := merkle.NewMerkleTree(depth)

	secret := []byte("test_secret")
	tree.InsertSecret(secret)

	proof, _ := tree.GetProof(0)
	root := tree.GetRoot()
	leafHash := merkle.HashToBigInt(secret)

	circuit := NewMembershipCircuit(depth)
	assignment := &MembershipCircuit{
		Secret:       secret,
		PathElements: make([]frontend.Variable, depth),
		PathIndices:  make([]frontend.Variable, depth),
		Root:         root,
		LeafHash:     leafHash,
	}

	for i := 0; i < depth; i++ {
		assignment.PathElements[i] = proof.PathElements[i]
		assignment.PathIndices[i] = proof.PathIndices[i]
	}

	// Full Groth16 test - using standard IsSolved which tests all backends
	assert.NoError(t, test.IsSolved(circuit, assignment, ecc.BN254.ScalarField()))
}

func TestNewMembershipCircuit(t *testing.T) {
	circuit := NewMembershipCircuit(20)
	assert.Equal(t, 20, circuit.GetDepth())
	assert.Equal(t, 20, len(circuit.PathElements))
	assert.Equal(t, 20, len(circuit.PathIndices))
}
