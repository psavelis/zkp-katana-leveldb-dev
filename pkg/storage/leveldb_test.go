package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempDBPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "test.db")
}

func TestNewLevelDBStore(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	assert.Equal(t, path, store.Path())
}

func TestLevelDBStore_PutGet(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	// Put
	key := []byte("testkey")
	value := []byte("testvalue")
	err = store.Put(key, value)
	require.NoError(t, err)

	// Get
	got, err := store.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, got)
}

func TestLevelDBStore_GetNotFound(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.Get([]byte("nonexistent"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestLevelDBStore_Delete(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	key := []byte("deletekey")
	store.Put(key, []byte("value"))

	err = store.Delete(key)
	require.NoError(t, err)

	_, err = store.Get(key)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestLevelDBStore_Has(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	key := []byte("haskey")

	has, err := store.Has(key)
	require.NoError(t, err)
	assert.False(t, has)

	store.Put(key, []byte("value"))

	has, err = store.Has(key)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestLevelDBStore_BatchWrite(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	ops := []BatchOp{
		{Key: []byte("batch1"), Value: []byte("value1")},
		{Key: []byte("batch2"), Value: []byte("value2")},
		{Key: []byte("batch3"), Value: []byte("value3")},
	}

	err = store.BatchWrite(ops)
	require.NoError(t, err)

	for _, op := range ops {
		val, err := store.Get(op.Key)
		require.NoError(t, err)
		assert.Equal(t, op.Value, val)
	}

	// Test delete in batch
	deleteOps := []BatchOp{
		{Key: []byte("batch1"), Delete: true},
	}
	err = store.BatchWrite(deleteOps)
	require.NoError(t, err)

	_, err = store.Get([]byte("batch1"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestLevelDBStore_ForEach(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	// Add some keys with prefix
	prefix := "test:"
	for i := 0; i < 5; i++ {
		key := []byte(prefix + string(rune('a'+i)))
		store.Put(key, []byte("value"))
	}

	// Add keys without prefix
	store.Put([]byte("other1"), []byte("value"))
	store.Put([]byte("other2"), []byte("value"))

	// Count keys with prefix
	count := 0
	err = store.ForEach([]byte(prefix), func(key, value []byte) error {
		count++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestLevelDBStore_Count(t *testing.T) {
	path := tempDBPath(t)
	store, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store.Close()

	prefix := "count:"
	for i := 0; i < 10; i++ {
		key := []byte(prefix + string(rune('a'+i)))
		store.Put(key, []byte("value"))
	}

	count, err := store.Count([]byte(prefix))
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestLevelDBStore_Persistence(t *testing.T) {
	path := tempDBPath(t)

	// Write and close
	store1, err := NewLevelDBStore(path)
	require.NoError(t, err)
	store1.Put([]byte("persistent"), []byte("data"))
	store1.Close()

	// Reopen and read
	store2, err := NewLevelDBStore(path)
	require.NoError(t, err)
	defer store2.Close()

	val, err := store2.Get([]byte("persistent"))
	require.NoError(t, err)
	assert.Equal(t, []byte("data"), val)
}

// ProofCache tests
func TestProofCache_StoreAndGet(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	cache := NewProofCache(store)

	proof := &StoredProof{
		ID:          "test-proof-1",
		Proof:       []byte("proof-data"),
		Root:        "abc123",
		LeafHash:    "def456",
		CreatedAt:   time.Now(),
		TreeDepth:   10,
		ProofTimeMs: 500,
	}

	err := cache.StoreProof("test-proof-1", proof)
	require.NoError(t, err)

	got, err := cache.GetProof("test-proof-1")
	require.NoError(t, err)
	assert.Equal(t, proof.Root, got.Root)
	assert.Equal(t, proof.LeafHash, got.LeafHash)
}

func TestProofCache_HasProof(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	cache := NewProofCache(store)

	has, _ := cache.HasProof("nonexistent")
	assert.False(t, has)

	cache.StoreProof("exists", &StoredProof{})

	has, _ = cache.HasProof("exists")
	assert.True(t, has)
}

func TestProofCache_ListProofs(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	cache := NewProofCache(store)

	ids := []string{"proof1", "proof2", "proof3"}
	for _, id := range ids {
		cache.StoreProof(id, &StoredProof{Root: id})
	}

	listed, err := cache.ListProofs()
	require.NoError(t, err)
	assert.Len(t, listed, 3)
}

func TestProofCache_ProofCount(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	cache := NewProofCache(store)

	for i := 0; i < 5; i++ {
		cache.StoreProof(string(rune('a'+i)), &StoredProof{})
	}

	count, err := cache.ProofCount()
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

// StateTrie tests
func TestStateTrie_UpdateRoot(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	trie := NewStateTrie(store)

	root1 := []byte("root1")
	err := trie.UpdateRoot(root1, 10)
	require.NoError(t, err)

	got, err := trie.GetCurrentRoot()
	require.NoError(t, err)
	assert.Equal(t, root1, got)
}

func TestStateTrie_RootHistory(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	trie := NewStateTrie(store)

	// Add multiple roots
	for i := 0; i < 5; i++ {
		root := []byte{byte(i)}
		trie.UpdateRoot(root, i*10)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	history, err := trie.GetRootHistory(10)
	require.NoError(t, err)
	assert.Len(t, history, 5)

	// Most recent should be first
	assert.Equal(t, 40, history[0].LeafCount)
}

func TestStateTrie_HasRoot(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	trie := NewStateTrie(store)

	root := []byte("testroot")
	trie.UpdateRoot(root, 5)

	has, err := trie.HasRoot(root)
	require.NoError(t, err)
	assert.True(t, has)

	has, err = trie.HasRoot([]byte("nonexistent"))
	require.NoError(t, err)
	assert.False(t, has)
}

func TestStateTrie_Persistence(t *testing.T) {
	path := tempDBPath(t)

	// Write root and close
	store1, _ := NewLevelDBStore(path)
	trie1 := NewStateTrie(store1)
	root := []byte("persistent-root")
	trie1.UpdateRoot(root, 100)
	store1.Close()

	// Reopen and verify
	store2, _ := NewLevelDBStore(path)
	defer store2.Close()
	trie2 := NewStateTrie(store2)

	got, err := trie2.GetCurrentRoot()
	require.NoError(t, err)
	assert.Equal(t, root, got)
}

func TestStateTrie_Clear(t *testing.T) {
	path := tempDBPath(t)
	store, _ := NewLevelDBStore(path)
	defer store.Close()

	trie := NewStateTrie(store)

	for i := 0; i < 3; i++ {
		trie.UpdateRoot([]byte{byte(i)}, i)
	}

	err := trie.Clear()
	require.NoError(t, err)

	_, err = trie.GetCurrentRoot()
	assert.ErrorIs(t, err, ErrNotFound)

	count, _ := trie.RootCount()
	assert.Equal(t, 0, count)
}

// Benchmark tests
func BenchmarkLevelDBStore_Put(b *testing.B) {
	path := filepath.Join(os.TempDir(), "bench.db")
	defer os.RemoveAll(path)

	store, _ := NewLevelDBStore(path)
	defer store.Close()

	value := []byte("benchmark-value-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := []byte(string(rune(i)))
		store.Put(key, value)
	}
}

func BenchmarkLevelDBStore_Get(b *testing.B) {
	path := filepath.Join(os.TempDir(), "bench.db")
	defer os.RemoveAll(path)

	store, _ := NewLevelDBStore(path)
	defer store.Close()

	key := []byte("benchmark-key")
	value := []byte("benchmark-value-data")
	store.Put(key, value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get(key)
	}
}

func BenchmarkLevelDBStore_BatchWrite(b *testing.B) {
	path := filepath.Join(os.TempDir(), "bench.db")
	defer os.RemoveAll(path)

	store, _ := NewLevelDBStore(path)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ops := make([]BatchOp, 100)
		for j := 0; j < 100; j++ {
			ops[j] = BatchOp{
				Key:   []byte(string(rune(i*100 + j))),
				Value: []byte("value"),
			}
		}
		store.BatchWrite(ops)
	}
}
