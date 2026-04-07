package katana

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:5050", client.config.RPCURL)
}

func TestNewClientWithURL(t *testing.T) {
	client, err := NewClientWithURL("http://custom:1234")
	require.NoError(t, err)
	assert.Equal(t, "http://custom:1234", client.config.RPCURL)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "http://localhost:5050", config.RPCURL)
	assert.Equal(t, "KATANA", config.ChainID)
}

func TestMockClient_GetBlockNumber(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	blockNum, err := mock.GetBlockNumber(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), blockNum)

	mock.SetBlockNumber(100)
	blockNum, err = mock.GetBlockNumber(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(100), blockNum)
}

func TestMockClient_AdvanceBlock(t *testing.T) {
	mock := NewMockClient()

	assert.Equal(t, uint64(2), mock.AdvanceBlock())
	assert.Equal(t, uint64(3), mock.AdvanceBlock())
}

func TestMockClient_GetChainID(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	chainID, err := mock.GetChainID(ctx)
	require.NoError(t, err)
	assert.Equal(t, "SN_KATANA", chainID)
}

func TestMockClient_GetChainStatus(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	status, err := mock.GetChainStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, "SN_KATANA", status.ChainID)
	assert.Equal(t, uint64(1), status.BlockNumber)
	assert.False(t, status.Syncing)
}

func TestMockClient_SimulateVerification(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	req := &VerificationRequest{
		Proof: Groth16Proof{
			A: G1Point{X: big.NewInt(1), Y: big.NewInt(2)},
			B: G2Point{X0: big.NewInt(3), X1: big.NewInt(4), Y0: big.NewInt(5), Y1: big.NewInt(6)},
			C: G1Point{X: big.NewInt(7), Y: big.NewInt(8)},
		},
		MerkleRoot: big.NewInt(12345),
		LeafHash:   big.NewInt(67890),
	}

	result, err := mock.SimulateVerification(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, uint64(100000), result.GasUsed)
}

func TestMockClient_SimulateVerification_Invalid(t *testing.T) {
	mock := NewMockClient()
	mock.SetSimulateValid(false)
	ctx := context.Background()

	req := &VerificationRequest{
		MerkleRoot: big.NewInt(111),
		LeafHash:   big.NewInt(222),
	}

	result, err := mock.SimulateVerification(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Valid)
}

func TestMockClient_SubmitProofCommitment(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	commitment := []byte("test-commitment-data")
	result, err := mock.SubmitProofCommitment(ctx, commitment)
	require.NoError(t, err)
	assert.NotEmpty(t, result.TransactionHash)
	assert.Equal(t, "ACCEPTED_ON_L2", result.Status)

	// Block should advance
	blockNum, _ := mock.GetBlockNumber(ctx)
	assert.Equal(t, uint64(2), blockNum)
}

func TestMockClient_GetVerifiedProofs(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	req := &VerificationRequest{
		MerkleRoot: big.NewInt(100),
		LeafHash:   big.NewInt(200),
	}

	mock.SimulateVerification(ctx, req)
	mock.SimulateVerification(ctx, req)

	proofs := mock.GetVerifiedProofs()
	assert.Len(t, proofs, 2)
}

func TestMockClient_IsProofVerified(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	root := big.NewInt(123)
	leaf := big.NewInt(456)

	// Not verified yet
	assert.False(t, mock.IsProofVerified(root, leaf))

	// Verify
	mock.SimulateVerification(ctx, &VerificationRequest{
		MerkleRoot: root,
		LeafHash:   leaf,
	})

	assert.True(t, mock.IsProofVerified(root, leaf))
}

func TestMockClient_Reset(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	mock.SetBlockNumber(100)
	mock.SimulateVerification(ctx, &VerificationRequest{MerkleRoot: big.NewInt(1)})

	mock.Reset()

	blockNum, _ := mock.GetBlockNumber(ctx)
	assert.Equal(t, uint64(1), blockNum)
	assert.Empty(t, mock.GetVerifiedProofs())
}

func TestToFelt(t *testing.T) {
	tests := []struct {
		input    *big.Int
		expected string
	}{
		{big.NewInt(0), "0x"},
		{big.NewInt(255), "0xff"},
		{big.NewInt(256), "0x0100"},
		{nil, "0x0"},
	}

	for _, tt := range tests {
		result := ToFelt(tt.input)
		if tt.input == nil {
			assert.Equal(t, "0x0", result)
		} else {
			assert.True(t, len(result) >= 2)
			assert.Equal(t, "0x", result[:2])
		}
	}
}

func TestFromFelt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0xff", 255},
		{"ff", 255},
		{"0x0100", 256},
		{"0x0", 0},
	}

	for _, tt := range tests {
		result, err := FromFelt(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, result.Int64())
	}
}

func TestToFeltArray(t *testing.T) {
	values := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	result := ToFeltArray(values)

	assert.Len(t, result, 3)
	for _, r := range result {
		assert.True(t, len(r) >= 2)
		assert.Equal(t, "0x", r[:2])
	}
}

func TestMockClient_Ping(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	err := mock.Ping(ctx)
	assert.NoError(t, err)
}

func TestMockClient_GetBlock(t *testing.T) {
	mock := NewMockClient()
	mock.SetBlockNumber(42)
	ctx := context.Background()

	block, err := mock.GetBlock(ctx, "latest")
	require.NoError(t, err)
	assert.Equal(t, uint64(42), block.BlockNumber)
	assert.NotEmpty(t, block.BlockHash)
	assert.NotEmpty(t, block.ParentHash)
}

func TestIsScarbAvailable(t *testing.T) {
	// Just ensure it doesn't panic
	_ = IsScarbAvailable()
}

func TestIsStarkliAvailable(t *testing.T) {
	// Just ensure it doesn't panic
	_ = IsStarkliAvailable()
}
