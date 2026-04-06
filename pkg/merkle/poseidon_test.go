package merkle

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoseidonHash_Hash(t *testing.T) {
	h := NewPoseidonHash()

	// Test basic hashing
	data := []byte("hello world")
	result := h.Hash(data)
	assert.NotNil(t, result)
	assert.Equal(t, 32, len(result), "hash should be 32 bytes")

	// Test deterministic
	result2 := h.Hash(data)
	assert.Equal(t, result, result2, "hash should be deterministic")

	// Test different inputs produce different hashes
	result3 := h.Hash([]byte("different data"))
	assert.NotEqual(t, result, result3, "different inputs should produce different hashes")
}

func TestPoseidonHash_HashPair(t *testing.T) {
	h := NewPoseidonHash()

	left := []byte("left")
	right := []byte("right")

	result := h.HashPair(left, right)
	assert.NotNil(t, result)
	assert.Equal(t, 32, len(result))

	// Order matters
	result2 := h.HashPair(right, left)
	assert.NotEqual(t, result, result2, "hash(left, right) != hash(right, left)")
}

func TestPoseidonHash_HashElements(t *testing.T) {
	h := NewPoseidonHash()

	elem1 := big.NewInt(12345)
	elem2 := big.NewInt(67890)

	result := h.HashElements(elem1, elem2)
	assert.NotNil(t, result)
	assert.True(t, result.Sign() > 0, "result should be positive")

	// Deterministic
	result2 := h.HashElements(elem1, elem2)
	assert.Equal(t, 0, result.Cmp(result2), "hash should be deterministic")
}

func TestHashBytes(t *testing.T) {
	data := []byte("test data")
	result := HashBytes(data)
	require.NotNil(t, result)
	assert.Equal(t, 32, len(result))
}

func TestHashPairBytes(t *testing.T) {
	left := []byte("left")
	right := []byte("right")
	result := HashPairBytes(left, right)
	require.NotNil(t, result)
	assert.Equal(t, 32, len(result))
}

func TestHashToBigInt(t *testing.T) {
	data := []byte("test")
	result := HashToBigInt(data)
	require.NotNil(t, result)
	assert.True(t, result.Sign() >= 0)
}

func TestEmptyLeaf(t *testing.T) {
	leaf := EmptyLeaf()
	assert.NotNil(t, leaf)
	assert.Equal(t, 32, len(leaf))

	// Should be deterministic
	leaf2 := EmptyLeaf()
	assert.Equal(t, leaf, leaf2)
}

func TestEmptyLeafBigInt(t *testing.T) {
	leaf := EmptyLeafBigInt()
	assert.NotNil(t, leaf)
	assert.True(t, leaf.Sign() >= 0)
}

func BenchmarkPoseidonHash(b *testing.B) {
	h := NewPoseidonHash()
	data := []byte("benchmark data for poseidon hash")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Hash(data)
	}
}

func BenchmarkHashPair(b *testing.B) {
	h := NewPoseidonHash()
	left := []byte("left data for benchmark")
	right := []byte("right data for benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.HashPair(left, right)
	}
}
