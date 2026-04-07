//go:build e2e
// +build e2e

// Package e2e provides end-to-end integration tests.
package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/katana"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/merkle"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/prover"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/storage"
)

const TestTreeDepth = 10

func TestE2E_FullFlow(t *testing.T) {
	ctx := context.Background()

	// Setup temporary storage
	dbPath := filepath.Join(t.TempDir(), "e2e-test.db")
	store, err := storage.NewLevelDBStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	proofCache := storage.NewProofCache(store)
	stateTrie := storage.NewStateTrie(store)

	// Create Merkle tree
	tree := merkle.NewMerkleTree(TestTreeDepth)

	// Add members
	members := []string{"alice", "bob", "charlie"}
	for _, m := range members {
		_, _, err := tree.InsertSecret([]byte(m))
		require.NoError(t, err)
	}

	root := tree.GetRoot()
	err = stateTrie.UpdateRoot(root.Bytes(), tree.Size())
	require.NoError(t, err)

	// Initialize prover
	zkProver, err := prover.NewProver(TestTreeDepth)
	require.NoError(t, err)

	// Generate proof for alice
	aliceSecret := []byte("alice")
	aliceLeafHash := merkle.HashToBigInt(aliceSecret)
	treeProof, err := tree.GetProof(0)
	require.NoError(t, err)

	witness := &prover.MembershipWitness{
		Secret:       aliceSecret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     aliceLeafHash,
	}

	// Generate proof
	start := time.Now()
	proofResult, err := zkProver.Prove(witness)
	proveTime := time.Since(start)
	require.NoError(t, err)

	t.Logf("Proof generated in %v", proveTime)

	// Verify proof
	start = time.Now()
	valid, err := zkProver.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
	verifyTime := time.Since(start)
	require.NoError(t, err)
	assert.True(t, valid)

	t.Logf("Proof verified in %v", verifyTime)

	// Store proof
	proofData := proofResult.ToProofData()
	err = proofCache.StoreProof("alice-membership", &storage.StoredProof{
		ID:          "alice-membership",
		Proof:       proofData.Proof,
		Root:        proofData.Root,
		LeafHash:    proofData.LeafHash,
		CreatedAt:   time.Now(),
		TreeDepth:   TestTreeDepth,
		ProofTimeMs: proofData.ProofTimeMs,
		Verified:    valid,
	})
	require.NoError(t, err)

	// Retrieve and verify stored proof
	storedProof, err := proofCache.GetProof("alice-membership")
	require.NoError(t, err)
	assert.Equal(t, proofData.Root, storedProof.Root)
	assert.True(t, storedProof.Verified)

	// Check total time
	totalTime := proveTime + verifyTime
	t.Logf("Total round-trip: %v", totalTime)

	// Performance assertion (informational)
	if totalTime > 2*time.Second {
		t.Logf("Warning: Round-trip exceeded 2 seconds (got %v)", totalTime)
	}
}

func TestE2E_KatanaIntegration(t *testing.T) {
	ctx := context.Background()

	// Get Katana URL from environment
	katanaURL := os.Getenv("KATANA_RPC_URL")
	if katanaURL == "" {
		katanaURL = "http://localhost:5050"
	}

	// Try to connect to Katana
	client, err := katana.NewClientWithURL(katanaURL)
	if err != nil {
		t.Skipf("Failed to create Katana client: %v", err)
	}

	// Check if Katana is available
	if err := client.Ping(ctx); err != nil {
		t.Skipf("Katana not available at %s: %v", katanaURL, err)
	}

	// Get chain status
	status, err := client.GetChainStatus(ctx)
	require.NoError(t, err)

	t.Logf("Connected to Katana: chain=%s, block=%d", status.ChainID, status.BlockNumber)

	// Test block retrieval
	block, err := client.GetLatestBlock(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, block.BlockHash)
}

func TestE2E_MockKatanaFlow(t *testing.T) {
	ctx := context.Background()

	// Setup components
	tree := merkle.NewMerkleTree(TestTreeDepth)
	secret := []byte("test_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	root := tree.GetRoot()

	// Create mock Katana client
	mock := katana.NewMockClient()

	// Verify chain is accessible
	blockNum, err := mock.GetBlockNumber(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), blockNum)

	// Simulate verification
	req := &katana.VerificationRequest{
		Proof: katana.Groth16Proof{
			A: katana.G1Point{X: root, Y: leafHash},
		},
		MerkleRoot: root,
		LeafHash:   leafHash,
	}

	result, err := mock.SimulateVerification(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Valid)

	// Submit commitment
	txResult, err := mock.SubmitProofCommitment(ctx, root.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "ACCEPTED_ON_L2", txResult.Status)

	// Verify block advanced
	blockNum, _ = mock.GetBlockNumber(ctx)
	assert.Equal(t, uint64(2), blockNum)
}

func TestE2E_MultipleMembers(t *testing.T) {
	// Setup
	tree := merkle.NewMerkleTree(TestTreeDepth)
	zkProver, err := prover.NewProver(TestTreeDepth)
	require.NoError(t, err)

	// Add many members
	memberCount := 10
	members := make([][]byte, memberCount)
	leafHashes := make([]interface{}, memberCount)

	for i := 0; i < memberCount; i++ {
		members[i] = []byte(string(rune('a' + i)))
		_, hash, err := tree.InsertSecret(members[i])
		require.NoError(t, err)
		leafHashes[i] = hash
	}

	root := tree.GetRoot()

	// Generate and verify proofs for all members
	for i := 0; i < memberCount; i++ {
		treeProof, err := tree.GetProof(i)
		require.NoError(t, err)

		witness := &prover.MembershipWitness{
			Secret:       members[i],
			PathElements: treeProof.PathElements,
			PathIndices:  treeProof.PathIndices,
			Root:         root,
			LeafHash:     merkle.HashToBigInt(members[i]),
		}

		proofResult, err := zkProver.Prove(witness)
		require.NoError(t, err)

		valid, err := zkProver.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
		require.NoError(t, err)
		assert.True(t, valid, "Proof for member %d should be valid", i)
	}
}

func TestE2E_StoragePersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persistence-test.db")

	// Write data
	store1, err := storage.NewLevelDBStore(dbPath)
	require.NoError(t, err)

	cache1 := storage.NewProofCache(store1)
	err = cache1.StoreProof("test-proof", &storage.StoredProof{
		ID:       "test-proof",
		Root:     "test-root",
		LeafHash: "test-leaf",
	})
	require.NoError(t, err)
	store1.Close()

	// Read data (new instance)
	store2, err := storage.NewLevelDBStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	cache2 := storage.NewProofCache(store2)
	proof, err := cache2.GetProof("test-proof")
	require.NoError(t, err)
	assert.Equal(t, "test-root", proof.Root)
}

func BenchmarkE2E_FullFlow(b *testing.B) {
	tree := merkle.NewMerkleTree(TestTreeDepth)
	secret := []byte("benchmark_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	zkProver, _ := prover.NewProver(TestTreeDepth)

	witness := &prover.MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proofResult, _ := zkProver.Prove(witness)
		zkProver.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
	}
}
