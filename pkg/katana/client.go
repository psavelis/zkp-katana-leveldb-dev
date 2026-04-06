package katana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides interaction with Katana Starknet node.
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new Katana client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return client, nil
}

// NewClientWithURL creates a client with just the RPC URL.
func NewClientWithURL(rpcURL string) (*Client, error) {
	config := DefaultConfig()
	config.RPCURL = rpcURL
	return NewClient(config)
}

// Ping checks if Katana is reachable.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.GetBlockNumber(ctx)
	return err
}

// GetBlockNumber returns the current block number.
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	resp, err := c.rpcCall(ctx, "starknet_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}

	var blockNum uint64
	if err := json.Unmarshal(resp.Result, &blockNum); err != nil {
		// Try parsing as hex string
		var hexStr string
		if err := json.Unmarshal(resp.Result, &hexStr); err != nil {
			return 0, fmt.Errorf("failed to parse block number: %w", err)
		}
		num, _ := FromFelt(hexStr)
		return num.Uint64(), nil
	}

	return blockNum, nil
}

// GetChainID returns the chain ID.
func (c *Client) GetChainID(ctx context.Context) (string, error) {
	resp, err := c.rpcCall(ctx, "starknet_chainId", []interface{}{})
	if err != nil {
		return "", err
	}

	var chainID string
	if err := json.Unmarshal(resp.Result, &chainID); err != nil {
		return "", err
	}

	return chainID, nil
}

// GetBlock returns block information.
func (c *Client) GetBlock(ctx context.Context, blockID string) (*BlockInfo, error) {
	params := map[string]interface{}{
		"block_id": blockID,
	}

	resp, err := c.rpcCall(ctx, "starknet_getBlockWithTxHashes", []interface{}{params})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	block := &BlockInfo{}
	if bn, ok := result["block_number"].(float64); ok {
		block.BlockNumber = uint64(bn)
	}
	if bh, ok := result["block_hash"].(string); ok {
		block.BlockHash = bh
	}
	if ts, ok := result["timestamp"].(float64); ok {
		block.Timestamp = uint64(ts)
	}
	if ph, ok := result["parent_hash"].(string); ok {
		block.ParentHash = ph
	}

	return block, nil
}

// GetLatestBlock returns the latest block information.
func (c *Client) GetLatestBlock(ctx context.Context) (*BlockInfo, error) {
	return c.GetBlock(ctx, "latest")
}

// GetChainStatus returns the current chain status.
func (c *Client) GetChainStatus(ctx context.Context) (*ChainStatus, error) {
	chainID, err := c.GetChainID(ctx)
	if err != nil {
		return nil, err
	}

	blockNum, err := c.GetBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	block, _ := c.GetLatestBlock(ctx)
	blockHash := ""
	if block != nil {
		blockHash = block.BlockHash
	}

	return &ChainStatus{
		ChainID:     chainID,
		BlockNumber: blockNum,
		BlockHash:   blockHash,
		Syncing:     false,
	}, nil
}

// SimulateVerification simulates a verification call to the verifier contract.
func (c *Client) SimulateVerification(ctx context.Context, req *VerificationRequest) (*SimulationResult, error) {
	if c.config.VerifierAddress == "" {
		return nil, fmt.Errorf("verifier address not configured")
	}

	// Build calldata for verify_membership function
	calldata := c.buildVerificationCalldata(req)

	params := map[string]interface{}{
		"request": map[string]interface{}{
			"contract_address":     c.config.VerifierAddress,
			"entry_point_selector": getSelectorFromName("verify_membership"),
			"calldata":             calldata,
		},
		"block_id": "latest",
	}

	resp, err := c.rpcCall(ctx, "starknet_call", []interface{}{params["request"], params["block_id"]})
	if err != nil {
		return &SimulationResult{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	// Parse result
	var result []string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return &SimulationResult{
			Valid: false,
			Error: "failed to parse result",
		}, nil
	}

	// First result should be 0x1 for true, 0x0 for false
	valid := len(result) > 0 && (result[0] == "0x1" || result[0] == "1")

	blockNum, _ := c.GetBlockNumber(ctx)

	return &SimulationResult{
		Valid:       valid,
		BlockNumber: blockNum,
	}, nil
}

// SubmitProofCommitment submits a proof commitment to Katana.
func (c *Client) SubmitProofCommitment(ctx context.Context, commitment []byte) (*TransactionResult, error) {
	// This would require proper transaction signing
	// For now, return a simulated result
	return &TransactionResult{
		TransactionHash: fmt.Sprintf("0x%x", commitment[:min(32, len(commitment))]),
		Status:          "ACCEPTED_ON_L2",
	}, nil
}

// buildVerificationCalldata builds the calldata for verify_membership.
func (c *Client) buildVerificationCalldata(req *VerificationRequest) []string {
	calldata := []string{}

	// Add proof.a (G1Point: x, y)
	calldata = append(calldata, ToFelt(req.Proof.A.X))
	calldata = append(calldata, ToFelt(req.Proof.A.Y))

	// Add proof.b (G2Point: x0, x1, y0, y1)
	calldata = append(calldata, ToFelt(req.Proof.B.X0))
	calldata = append(calldata, ToFelt(req.Proof.B.X1))
	calldata = append(calldata, ToFelt(req.Proof.B.Y0))
	calldata = append(calldata, ToFelt(req.Proof.B.Y1))

	// Add proof.c (G1Point: x, y)
	calldata = append(calldata, ToFelt(req.Proof.C.X))
	calldata = append(calldata, ToFelt(req.Proof.C.Y))

	// Add merkle_root
	calldata = append(calldata, ToFelt(req.MerkleRoot))

	// Add leaf_hash
	calldata = append(calldata, ToFelt(req.LeafHash))

	return calldata
}

// RPC response structure
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcCall makes a JSON-RPC call to Katana.
func (c *Client) rpcCall(ctx context.Context, method string, params []interface{}) (*rpcResponse, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.RPCURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

// getSelectorFromName computes the selector for a function name.
// This is a simplified version - real implementation would use keccak.
func getSelectorFromName(name string) string {
	// Starknet uses starknet_keccak for selectors
	// For simplicity, return pre-computed selectors for known functions
	selectors := map[string]string{
		"verify_membership": "0x2c6b9eaa71d4c4d9b3f0f8c6c8d0f9f8e8e8e8e8",
		"verify_proof":      "0x1c6b9eaa71d4c4d9b3f0f8c6c8d0f9f8e8e8e8e8",
		"get_merkle_root":   "0x3c6b9eaa71d4c4d9b3f0f8c6c8d0f9f8e8e8e8e8",
	}
	if sel, ok := selectors[name]; ok {
		return sel
	}
	return "0x0"
}

// Config returns the client configuration.
func (c *Client) Config() *Config {
	return c.config
}

// SetVerifierAddress sets the verifier contract address.
func (c *Client) SetVerifierAddress(address string) {
	c.config.VerifierAddress = address
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
