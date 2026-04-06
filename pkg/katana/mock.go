package katana

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// MockClient provides a mock Katana client for testing without a real node.
type MockClient struct {
	mu              sync.RWMutex
	blockNumber     uint64
	chainID         string
	proofs          map[string]bool
	verifiedProofs  []*VerificationRequest
	simulateValid   bool
	simulateError   error
	commitments     [][]byte
	verifierAddress string
}

// NewMockClient creates a new mock Katana client.
func NewMockClient() *MockClient {
	return &MockClient{
		blockNumber:   1,
		chainID:       "SN_KATANA",
		proofs:        make(map[string]bool),
		simulateValid: true,
	}
}

// Ping always succeeds for mock.
func (m *MockClient) Ping(ctx context.Context) error {
	return nil
}

// GetBlockNumber returns the current mock block number.
func (m *MockClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blockNumber, nil
}

// GetChainID returns the mock chain ID.
func (m *MockClient) GetChainID(ctx context.Context) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chainID, nil
}

// GetBlock returns mock block information.
func (m *MockClient) GetBlock(ctx context.Context, blockID string) (*BlockInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &BlockInfo{
		BlockNumber: m.blockNumber,
		BlockHash:   fmt.Sprintf("0x%064x", m.blockNumber),
		Timestamp:   uint64(time.Now().Unix()),
		ParentHash:  fmt.Sprintf("0x%064x", m.blockNumber-1),
	}, nil
}

// GetLatestBlock returns the latest mock block.
func (m *MockClient) GetLatestBlock(ctx context.Context) (*BlockInfo, error) {
	return m.GetBlock(ctx, "latest")
}

// GetChainStatus returns mock chain status.
func (m *MockClient) GetChainStatus(ctx context.Context) (*ChainStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &ChainStatus{
		ChainID:     m.chainID,
		BlockNumber: m.blockNumber,
		BlockHash:   fmt.Sprintf("0x%064x", m.blockNumber),
		Syncing:     false,
	}, nil
}

// SimulateVerification simulates verification and returns configured result.
func (m *MockClient) SimulateVerification(ctx context.Context, req *VerificationRequest) (*SimulationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.simulateError != nil {
		return nil, m.simulateError
	}

	// Store verified proof
	m.verifiedProofs = append(m.verifiedProofs, req)

	// Mark proof as verified
	proofKey := fmt.Sprintf("%s:%s", ToFelt(req.MerkleRoot), ToFelt(req.LeafHash))
	m.proofs[proofKey] = m.simulateValid

	return &SimulationResult{
		Valid:       m.simulateValid,
		GasUsed:     100000,
		BlockNumber: m.blockNumber,
	}, nil
}

// SubmitProofCommitment stores a commitment in mock storage.
func (m *MockClient) SubmitProofCommitment(ctx context.Context, commitment []byte) (*TransactionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.commitments = append(m.commitments, commitment)
	m.blockNumber++

	return &TransactionResult{
		TransactionHash: fmt.Sprintf("0x%x", commitment[:min(32, len(commitment))]),
		BlockNumber:     m.blockNumber,
		Status:          "ACCEPTED_ON_L2",
	}, nil
}

// Mock-specific methods for testing

// SetBlockNumber sets the mock block number.
func (m *MockClient) SetBlockNumber(n uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockNumber = n
}

// AdvanceBlock increments the block number.
func (m *MockClient) AdvanceBlock() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockNumber++
	return m.blockNumber
}

// SetSimulateValid configures whether simulations return valid.
func (m *MockClient) SetSimulateValid(valid bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateValid = valid
}

// SetSimulateError configures an error to return from simulations.
func (m *MockClient) SetSimulateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateError = err
}

// GetVerifiedProofs returns all verified proofs.
func (m *MockClient) GetVerifiedProofs() []*VerificationRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.verifiedProofs
}

// GetCommitments returns all submitted commitments.
func (m *MockClient) GetCommitments() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.commitments
}

// IsProofVerified checks if a specific proof was verified.
func (m *MockClient) IsProofVerified(root, leafHash *big.Int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	proofKey := fmt.Sprintf("%s:%s", ToFelt(root), ToFelt(leafHash))
	return m.proofs[proofKey]
}

// Reset clears all mock state.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blockNumber = 1
	m.proofs = make(map[string]bool)
	m.verifiedProofs = nil
	m.commitments = nil
	m.simulateValid = true
	m.simulateError = nil
}

// SetVerifierAddress sets the mock verifier address.
func (m *MockClient) SetVerifierAddress(address string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verifierAddress = address
}

// Config returns a mock config.
func (m *MockClient) Config() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &Config{
		RPCURL:          "http://mock:5050",
		ChainID:         m.chainID,
		VerifierAddress: m.verifierAddress,
	}
}
