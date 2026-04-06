// Package storage provides LevelDB-based persistent storage for ZK proofs and state.
package storage

import (
	"encoding/json"
	"time"
)

// Key prefixes for different data types.
const (
	PrefixProof      = "proof:"
	PrefixCommitment = "commitment:"
	PrefixStateRoot  = "stateroot:"
	PrefixMetadata   = "meta:"
)

// BatchOp represents a single operation in a batch write.
type BatchOp struct {
	Key    []byte
	Value  []byte
	Delete bool
}

// StoredProof represents a proof stored in the database.
type StoredProof struct {
	ID           string    `json:"id"`
	Proof        []byte    `json:"proof"`
	Root         string    `json:"root"`
	LeafHash     string    `json:"leaf_hash"`
	PublicInputs []string  `json:"public_inputs"`
	CreatedAt    time.Time `json:"created_at"`
	TreeDepth    int       `json:"tree_depth"`
	ProofTimeMs  int64     `json:"proof_time_ms"`
	Verified     bool      `json:"verified"`
}

// ToJSON serializes the stored proof.
func (p *StoredProof) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// StoredProofFromJSON deserializes a stored proof.
func StoredProofFromJSON(data []byte) (*StoredProof, error) {
	var p StoredProof
	err := json.Unmarshal(data, &p)
	return &p, err
}

// Commitment represents a stored commitment.
type Commitment struct {
	ID        string    `json:"id"`
	Value     []byte    `json:"value"`
	LeafIndex int       `json:"leaf_index"`
	TreeRoot  string    `json:"tree_root"`
	CreatedAt time.Time `json:"created_at"`
}

// ToJSON serializes the commitment.
func (c *Commitment) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// CommitmentFromJSON deserializes a commitment.
func CommitmentFromJSON(data []byte) (*Commitment, error) {
	var c Commitment
	err := json.Unmarshal(data, &c)
	return &c, err
}

// StateRootEntry represents a historical state root.
type StateRootEntry struct {
	Root      string    `json:"root"`
	Timestamp time.Time `json:"timestamp"`
	LeafCount int       `json:"leaf_count"`
	BlockNum  uint64    `json:"block_num,omitempty"`
}

// ToJSON serializes the state root entry.
func (s *StateRootEntry) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// StateRootEntryFromJSON deserializes a state root entry.
func StateRootEntryFromJSON(data []byte) (*StateRootEntry, error) {
	var s StateRootEntry
	err := json.Unmarshal(data, &s)
	return &s, err
}

// StoreMetadata contains metadata about the store.
type StoreMetadata struct {
	TreeDepth   int       `json:"tree_depth"`
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
	ProofCount  int64     `json:"proof_count"`
	LeafCount   int64     `json:"leaf_count"`
}

// ToJSON serializes store metadata.
func (m *StoreMetadata) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// StoreMetadataFromJSON deserializes store metadata.
func StoreMetadataFromJSON(data []byte) (*StoreMetadata, error) {
	var m StoreMetadata
	err := json.Unmarshal(data, &m)
	return &m, err
}
