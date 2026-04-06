package merkle

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMerkleTree(t *testing.T) {
	tree := NewMerkleTree(10)
	assert.NotNil(t, tree)
	assert.Equal(t, 10, tree.Depth())
	assert.Equal(t, 0, tree.Size())
	assert.Equal(t, 1024, tree.Capacity()) // 2^10
}

func TestMerkleTree_DefaultDepth(t *testing.T) {
	tree := NewMerkleTree(0)
	assert.Equal(t, TreeDepth, tree.Depth())
}

func TestMerkleTree_Insert(t *testing.T) {
	tree := NewMerkleTree(10)

	// Insert a leaf
	leaf := big.NewInt(12345)
	idx, err := tree.Insert(leaf)
	require.NoError(t, err)
	assert.Equal(t, 0, idx)
	assert.Equal(t, 1, tree.Size())

	// Insert another leaf
	leaf2 := big.NewInt(67890)
	idx2, err := tree.Insert(leaf2)
	require.NoError(t, err)
	assert.Equal(t, 1, idx2)
	assert.Equal(t, 2, tree.Size())
}

func TestMerkleTree_InsertBytes(t *testing.T) {
	tree := NewMerkleTree(10)

	idx, err := tree.InsertBytes([]byte("test data"))
	require.NoError(t, err)
	assert.Equal(t, 0, idx)
}

func TestMerkleTree_InsertSecret(t *testing.T) {
	tree := NewMerkleTree(10)

	secret := []byte("my_secret_value")
	idx, leafHash, err := tree.InsertSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, 0, idx)
	assert.NotNil(t, leafHash)

	// Verify the leaf hash matches what we'd get from hashing the secret
	expectedHash := HashToBigInt(secret)
	assert.Equal(t, 0, expectedHash.Cmp(leafHash))
}

func TestMerkleTree_GetLeaf(t *testing.T) {
	tree := NewMerkleTree(10)

	leaf := big.NewInt(12345)
	tree.Insert(leaf)

	// Get existing leaf
	got, err := tree.GetLeaf(0)
	require.NoError(t, err)
	assert.Equal(t, 0, leaf.Cmp(got))

	// Get non-existent leaf (out of range)
	_, err = tree.GetLeaf(5)
	assert.Error(t, err)
}

func TestMerkleTree_GetRoot(t *testing.T) {
	tree := NewMerkleTree(10)

	// Empty tree should have a valid root (all zeros)
	root := tree.GetRoot()
	assert.NotNil(t, root)

	// Insert leaves and verify root changes
	tree.Insert(big.NewInt(1))
	root1 := tree.GetRoot()
	assert.NotEqual(t, 0, root1.Cmp(root), "root should change after insert")

	tree.Insert(big.NewInt(2))
	root2 := tree.GetRoot()
	assert.NotEqual(t, 0, root2.Cmp(root1), "root should change after second insert")
}

func TestMerkleTree_GetProof(t *testing.T) {
	tree := NewMerkleTree(10)

	// Insert some leaves
	secret1 := []byte("secret1")
	secret2 := []byte("secret2")
	secret3 := []byte("secret3")

	idx1, leaf1, _ := tree.InsertSecret(secret1)
	tree.InsertSecret(secret2)
	tree.InsertSecret(secret3)

	// Get proof for first leaf
	proof, err := tree.GetProof(idx1)
	require.NoError(t, err)
	assert.NotNil(t, proof)
	assert.Equal(t, 10, len(proof.PathElements))
	assert.Equal(t, 10, len(proof.PathIndices))
	assert.Equal(t, idx1, proof.LeafIndex)

	// Verify the proof
	root := tree.GetRoot()
	valid := tree.VerifyProof(leaf1, root, proof)
	assert.True(t, valid, "proof should be valid")
}

func TestMerkleTree_VerifyProof(t *testing.T) {
	tree := NewMerkleTree(10)

	// Insert leaves
	secrets := []string{"alice", "bob", "charlie", "david", "eve"}
	leaves := make([]*big.Int, len(secrets))
	indices := make([]int, len(secrets))

	for i, secret := range secrets {
		idx, leaf, err := tree.InsertSecret([]byte(secret))
		require.NoError(t, err)
		leaves[i] = leaf
		indices[i] = idx
	}

	root := tree.GetRoot()

	// Verify each leaf's proof
	for i := range secrets {
		proof, err := tree.GetProof(indices[i])
		require.NoError(t, err)

		valid := tree.VerifyProof(leaves[i], root, proof)
		assert.True(t, valid, "proof for leaf %d should be valid", i)
	}

	// Invalid proof (wrong leaf)
	proof, _ := tree.GetProof(0)
	valid := tree.VerifyProof(leaves[1], root, proof)
	assert.False(t, valid, "proof with wrong leaf should be invalid")

	// Invalid proof (wrong root)
	wrongRoot := big.NewInt(999999)
	valid = tree.VerifyProof(leaves[0], wrongRoot, proof)
	assert.False(t, valid, "proof with wrong root should be invalid")
}

func TestVerifyMerkleProof_Standalone(t *testing.T) {
	tree := NewMerkleTree(10)

	secret := []byte("test_secret")
	_, leaf, _ := tree.InsertSecret(secret)
	root := tree.GetRoot()
	proof, _ := tree.GetProof(0)

	// Standalone verification
	valid := VerifyMerkleProof(leaf, root, proof)
	assert.True(t, valid)
}

func TestMerkleTree_TreeFull(t *testing.T) {
	// Use small tree for this test
	tree := NewMerkleTree(3) // Only 8 leaves

	// Fill the tree
	for i := 0; i < 8; i++ {
		_, err := tree.Insert(big.NewInt(int64(i)))
		require.NoError(t, err)
	}

	// Tree should be full
	assert.Equal(t, 8, tree.Size())
	assert.Equal(t, 8, tree.Capacity())

	// Next insert should fail
	_, err := tree.Insert(big.NewInt(999))
	assert.Error(t, err)
}

func TestMerkleTree_GetZeros(t *testing.T) {
	tree := NewMerkleTree(10)
	zeros := tree.GetZeros()

	assert.Equal(t, 11, len(zeros)) // depth + 1
	assert.NotNil(t, zeros[0])

	// Verify zeros are computed correctly
	hasher := NewPoseidonHash()
	for i := 1; i < len(zeros); i++ {
		expected := hasher.HashElements(zeros[i-1], zeros[i-1])
		assert.Equal(t, 0, expected.Cmp(zeros[i]), "zero[%d] should be hash of zero[%d], zero[%d]", i, i-1, i-1)
	}
}

func TestMerkleTree_DeterministicRoot(t *testing.T) {
	// Two trees with same data should have same root
	tree1 := NewMerkleTree(10)
	tree2 := NewMerkleTree(10)

	data := []string{"a", "b", "c", "d", "e"}
	for _, d := range data {
		tree1.InsertSecret([]byte(d))
		tree2.InsertSecret([]byte(d))
	}

	root1 := tree1.GetRoot()
	root2 := tree2.GetRoot()

	assert.Equal(t, 0, root1.Cmp(root2), "same data should produce same root")
}

func BenchmarkMerkleTree_Insert(b *testing.B) {
	tree := NewMerkleTree(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Insert(big.NewInt(int64(i)))
	}
}

func BenchmarkMerkleTree_GetRoot(b *testing.B) {
	tree := NewMerkleTree(20)

	// Insert some leaves
	for i := 0; i < 100; i++ {
		tree.Insert(big.NewInt(int64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GetRoot()
	}
}

func BenchmarkMerkleTree_GetProof(b *testing.B) {
	tree := NewMerkleTree(20)

	// Insert some leaves
	for i := 0; i < 100; i++ {
		tree.Insert(big.NewInt(int64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GetProof(i % 100)
	}
}

func BenchmarkMerkleTree_VerifyProof(b *testing.B) {
	tree := NewMerkleTree(20)

	leaf := big.NewInt(12345)
	tree.Insert(leaf)
	root := tree.GetRoot()
	proof, _ := tree.GetProof(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.VerifyProof(leaf, root, proof)
	}
}
