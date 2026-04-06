package prover

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/merkle"
)

func TestNewProver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping prover test in short mode")
	}

	prover, err := NewProver(10)
	require.NoError(t, err)
	assert.NotNil(t, prover)
	assert.True(t, prover.IsInitialized())
	assert.Equal(t, 10, prover.Depth())
	assert.Greater(t, prover.ConstraintCount(), 0)
}

func TestProver_ProveAndVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping prover test in short mode")
	}

	depth := 10
	prover, err := NewProver(depth)
	require.NoError(t, err)

	// Create Merkle tree and add members
	tree := merkle.NewMerkleTree(depth)
	secret := []byte("alice_secret")
	_, leafHash, err := tree.InsertSecret(secret)
	require.NoError(t, err)

	tree.InsertSecret([]byte("bob_secret"))
	tree.InsertSecret([]byte("charlie_secret"))

	// Get proof from tree
	treeProof, err := tree.GetProof(0)
	require.NoError(t, err)

	root := tree.GetRoot()

	// Create witness
	witness := &MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	// Generate proof
	result, err := prover.Prove(witness)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ProofBytes)
	assert.NotNil(t, result.Proof)

	t.Logf("Proof generated in %v", result.Duration)
	t.Logf("Proof size: %d bytes", len(result.ProofBytes))

	// Verify proof
	valid, err := prover.Verify(result.ProofBytes, result.PublicInputs)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestProver_InvalidProof(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping prover test in short mode")
	}

	depth := 10
	prover, err := NewProver(depth)
	require.NoError(t, err)

	// Create valid proof first
	tree := merkle.NewMerkleTree(depth)
	secret := []byte("test_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	result, err := prover.Prove(witness)
	require.NoError(t, err)

	// Verify with wrong root
	wrongInputs := NewPublicInputsFromBigInt(
		leafHash, // Use leafHash as root (wrong)
		leafHash,
	)
	valid, err := prover.Verify(result.ProofBytes, wrongInputs)
	require.NoError(t, err)
	assert.False(t, valid, "proof should be invalid with wrong root")
}

func TestProver_ExportVerificationKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping prover test in short mode")
	}

	prover, err := NewProver(10)
	require.NoError(t, err)

	vkBytes, err := prover.ExportVerificationKey()
	require.NoError(t, err)
	assert.NotEmpty(t, vkBytes)

	t.Logf("Verification key size: %d bytes", len(vkBytes))
}

func TestProver_GetKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping prover test in short mode")
	}

	prover, err := NewProver(10)
	require.NoError(t, err)

	vk, err := prover.GetVerificationKey()
	require.NoError(t, err)
	assert.NotNil(t, vk)

	pk, err := prover.GetProvingKey()
	require.NoError(t, err)
	assert.NotNil(t, pk)
}

func TestProver_NotInitialized(t *testing.T) {
	prover := &Prover{depth: 10}

	_, err := prover.Prove(&MembershipWitness{})
	assert.Error(t, err)

	_, err = prover.Verify([]byte{}, &PublicInputs{})
	assert.Error(t, err)
}

func TestVerifier_FromProver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping verifier test in short mode")
	}

	depth := 10

	// Setup prover and generate proof
	prover, err := NewProver(depth)
	require.NoError(t, err)

	tree := merkle.NewMerkleTree(depth)
	secret := []byte("verifier_test_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	proofResult, err := prover.Prove(witness)
	require.NoError(t, err)

	// Create standalone verifier from exported key
	vkBytes, err := prover.ExportVerificationKey()
	require.NoError(t, err)

	verifier, err := NewVerifier(vkBytes, depth)
	require.NoError(t, err)

	// Verify using standalone verifier
	valid, err := verifier.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifier_FromKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping verifier test in short mode")
	}

	depth := 10
	prover, err := NewProver(depth)
	require.NoError(t, err)

	vk, err := prover.GetVerificationKey()
	require.NoError(t, err)

	verifier := NewVerifierFromKey(vk, depth)
	assert.NotNil(t, verifier)
	assert.Equal(t, depth, verifier.Depth())
}

func TestProofResult_ToProofData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping proof data test in short mode")
	}

	depth := 10
	prover, err := NewProver(depth)
	require.NoError(t, err)

	tree := merkle.NewMerkleTree(depth)
	secret := []byte("test")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	result, err := prover.Prove(witness)
	require.NoError(t, err)

	proofData := result.ToProofData()
	assert.NotNil(t, proofData)
	assert.NotEmpty(t, proofData.Proof)
	assert.NotEmpty(t, proofData.Root)
	assert.NotEmpty(t, proofData.LeafHash)
	assert.False(t, proofData.CreatedAt.IsZero())
	assert.Greater(t, proofData.ProofTimeMs, int64(0))

	// Test JSON serialization
	jsonData, err := proofData.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test deserialization
	parsed, err := ProofDataFromJSON(jsonData)
	require.NoError(t, err)
	assert.Equal(t, proofData.Root, parsed.Root)
	assert.Equal(t, proofData.LeafHash, parsed.LeafHash)
}

func BenchmarkProver_Setup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewProver(10)
	}
}

func BenchmarkProver_Prove(b *testing.B) {
	depth := 10
	prover, _ := NewProver(depth)

	tree := merkle.NewMerkleTree(depth)
	secret := []byte("benchmark_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prover.Prove(witness)
	}
}

func BenchmarkProver_Verify(b *testing.B) {
	depth := 10
	prover, _ := NewProver(depth)

	tree := merkle.NewMerkleTree(depth)
	secret := []byte("benchmark_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	result, _ := prover.Prove(witness)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prover.Verify(result.ProofBytes, result.PublicInputs)
	}
}
