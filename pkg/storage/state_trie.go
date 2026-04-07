package storage

import (
	"encoding/hex"
	"fmt"
	"time"
)

// StateTrie manages Merkle tree state persistence.
type StateTrie struct {
	store       *LevelDBStore
	currentRoot []byte
}

// NewStateTrie creates a new state trie manager.
func NewStateTrie(store *LevelDBStore) *StateTrie {
	st := &StateTrie{store: store}
	// Try to load current root from storage
	if root, err := st.loadCurrentRoot(); err == nil {
		st.currentRoot = root
	}
	return st
}

// UpdateRoot stores a new root and maintains history.
func (t *StateTrie) UpdateRoot(newRoot []byte, leafCount int) error {
	// Store historical root with timestamp
	entry := &StateRootEntry{
		Root:      hex.EncodeToString(newRoot),
		Timestamp: time.Now(),
		LeafCount: leafCount,
	}

	data, err := entry.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize state root: %w", err)
	}

	// Create history key with timestamp for ordering
	historyKey := fmt.Sprintf("%s%d", PrefixStateRoot, time.Now().UnixNano())
	if err := t.store.Put([]byte(historyKey), data); err != nil {
		return err
	}

	// Update current root
	t.currentRoot = newRoot
	return t.store.Put([]byte(PrefixMetadata+"current_root"), newRoot)
}

// GetCurrentRoot returns the current Merkle root.
func (t *StateTrie) GetCurrentRoot() ([]byte, error) {
	if t.currentRoot != nil {
		return t.currentRoot, nil
	}
	return t.loadCurrentRoot()
}

// GetCurrentRootHex returns the current root as hex string.
func (t *StateTrie) GetCurrentRootHex() (string, error) {
	root, err := t.GetCurrentRoot()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(root), nil
}

// loadCurrentRoot loads the current root from storage.
func (t *StateTrie) loadCurrentRoot() ([]byte, error) {
	return t.store.Get([]byte(PrefixMetadata + "current_root"))
}

// GetRootHistory returns historical root entries.
func (t *StateTrie) GetRootHistory(limit int) ([]*StateRootEntry, error) {
	prefix := []byte(PrefixStateRoot)
	var entries []*StateRootEntry

	iter := t.store.NewIterator(prefix)
	defer iter.Release()

	// Iterate in reverse order (most recent first)
	var keys [][]byte
	var values [][]byte
	for iter.Next() {
		keys = append(keys, append([]byte{}, iter.Key()...))
		values = append(values, append([]byte{}, iter.Value()...))
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	// Process in reverse order
	count := 0
	for i := len(values) - 1; i >= 0 && count < limit; i-- {
		entry, err := StateRootEntryFromJSON(values[i])
		if err != nil {
			continue // Skip invalid entries
		}
		entries = append(entries, entry)
		count++
	}

	return entries, nil
}

// GetRootAtTime returns the root that was active at the given time.
func (t *StateTrie) GetRootAtTime(targetTime time.Time) (*StateRootEntry, error) {
	prefix := []byte(PrefixStateRoot)
	var closest *StateRootEntry

	err := t.store.ForEach(prefix, func(_, value []byte) error {
		entry, err := StateRootEntryFromJSON(value)
		if err != nil {
			return nil
		}
		if entry.Timestamp.Before(targetTime) || entry.Timestamp.Equal(targetTime) {
			if closest == nil || entry.Timestamp.After(closest.Timestamp) {
				closest = entry
			}
		}
		return nil
	})

	if closest == nil {
		return nil, ErrNotFound
	}

	return closest, err
}

// HasRoot checks if a root exists in history.
func (t *StateTrie) HasRoot(root []byte) (bool, error) {
	rootHex := hex.EncodeToString(root)
	prefix := []byte(PrefixStateRoot)
	found := false

	err := t.store.ForEach(prefix, func(_, value []byte) error {
		entry, err := StateRootEntryFromJSON(value)
		if err != nil {
			return nil
		}
		if entry.Root == rootHex {
			found = true
			return fmt.Errorf("found")
		}
		return nil
	})

	if err != nil && err.Error() == "found" {
		return true, nil
	}

	return found, nil
}

// RootCount returns the number of historical roots.
func (t *StateTrie) RootCount() (int, error) {
	return t.store.Count([]byte(PrefixStateRoot))
}

// Clear removes all state trie data.
func (t *StateTrie) Clear() error {
	// Delete current root
	if err := t.store.Delete([]byte(PrefixMetadata + "current_root")); err != nil && err != ErrNotFound {
		return err
	}

	// Delete all history
	prefix := []byte(PrefixStateRoot)
	var keysToDelete [][]byte

	err := t.store.ForEach(prefix, func(key, _ []byte) error {
		keysToDelete = append(keysToDelete, append([]byte{}, key...))
		return nil
	})
	if err != nil {
		return err
	}

	var ops []BatchOp
	for _, key := range keysToDelete {
		ops = append(ops, BatchOp{Key: key, Delete: true})
	}

	if len(ops) > 0 {
		if err := t.store.BatchWrite(ops); err != nil {
			return err
		}
	}

	t.currentRoot = nil
	return nil
}
