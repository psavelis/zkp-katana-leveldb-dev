package storage

import (
	"encoding/hex"
	"fmt"
	"time"
)

// ProofCache provides proof storage and retrieval operations.
type ProofCache struct {
	store *LevelDBStore
}

// NewProofCache creates a new proof cache using the given store.
func NewProofCache(store *LevelDBStore) *ProofCache {
	return &ProofCache{store: store}
}

// StoreProof stores a proof with the given ID.
func (c *ProofCache) StoreProof(id string, proof *StoredProof) error {
	key := c.proofKey(id)
	proof.ID = id

	data, err := proof.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize proof: %w", err)
	}

	return c.store.Put(key, data)
}

// GetProof retrieves a proof by ID.
func (c *ProofCache) GetProof(id string) (*StoredProof, error) {
	key := c.proofKey(id)

	data, err := c.store.Get(key)
	if err != nil {
		return nil, err
	}

	return StoredProofFromJSON(data)
}

// HasProof checks if a proof exists.
func (c *ProofCache) HasProof(id string) (bool, error) {
	key := c.proofKey(id)
	return c.store.Has(key)
}

// DeleteProof removes a proof.
func (c *ProofCache) DeleteProof(id string) error {
	key := c.proofKey(id)
	return c.store.Delete(key)
}

// ListProofs returns all proof IDs.
func (c *ProofCache) ListProofs() ([]string, error) {
	prefix := []byte(PrefixProof)
	var ids []string

	err := c.store.ForEach(prefix, func(key, _ []byte) error {
		id := string(key[len(prefix):])
		ids = append(ids, id)
		return nil
	})

	return ids, err
}

// GetProofByRoot retrieves a proof by its Merkle root.
func (c *ProofCache) GetProofByRoot(root []byte) (*StoredProof, error) {
	rootHex := hex.EncodeToString(root)
	prefix := []byte(PrefixProof)

	var found *StoredProof
	err := c.store.ForEach(prefix, func(_, value []byte) error {
		proof, err := StoredProofFromJSON(value)
		if err != nil {
			return nil // Skip invalid entries
		}
		if proof.Root == rootHex {
			found = proof
			return fmt.Errorf("found") // Stop iteration
		}
		return nil
	})

	if found != nil {
		return found, nil
	}
	if err != nil && err.Error() == "found" {
		return found, nil
	}

	return nil, ErrNotFound
}

// GetProofsByTimeRange returns proofs created within the given time range.
func (c *ProofCache) GetProofsByTimeRange(start, end time.Time) ([]*StoredProof, error) {
	prefix := []byte(PrefixProof)
	var proofs []*StoredProof

	err := c.store.ForEach(prefix, func(_, value []byte) error {
		proof, err := StoredProofFromJSON(value)
		if err != nil {
			return nil // Skip invalid
		}
		if proof.CreatedAt.After(start) && proof.CreatedAt.Before(end) {
			proofs = append(proofs, proof)
		}
		return nil
	})

	return proofs, err
}

// ProofCount returns the total number of stored proofs.
func (c *ProofCache) ProofCount() (int, error) {
	return c.store.Count([]byte(PrefixProof))
}

// StoreCommitment stores a commitment.
func (c *ProofCache) StoreCommitment(id string, commitment *Commitment) error {
	key := c.commitmentKey(id)
	commitment.ID = id

	data, err := commitment.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize commitment: %w", err)
	}

	return c.store.Put(key, data)
}

// GetCommitment retrieves a commitment by ID.
func (c *ProofCache) GetCommitment(id string) (*Commitment, error) {
	key := c.commitmentKey(id)

	data, err := c.store.Get(key)
	if err != nil {
		return nil, err
	}

	return CommitmentFromJSON(data)
}

// Helper functions for key construction
func (c *ProofCache) proofKey(id string) []byte {
	return []byte(PrefixProof + id)
}

func (c *ProofCache) commitmentKey(id string) []byte {
	return []byte(PrefixCommitment + id)
}
