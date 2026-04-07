// Demo application for ZK-SNARK Development Environment
//
// This demonstrates the complete flow:
// 1. Create a Merkle tree with members
// 2. Generate a membership proof using gnark/Groth16
// 3. Store the proof in LevelDB
// 4. Verify the proof locally
// 5. Connect to Katana for on-chain verification simulation
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/katana"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/merkle"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/prover"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/storage"
)

const (
	TreeDepth     = 10 // Using 10 for faster demo, use 20 for production
	DefaultDBPath = "./data/demo.db"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║   ZK-SNARK Development Environment Demo                        ║")
	fmt.Println("║   gnark + LevelDB + Katana                                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	ctx := context.Background()
	totalStart := time.Now()

	// Get database path from environment or use default
	dbPath := os.Getenv("LEVELDB_PATH")
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Printf("❌ Failed to create data directory: %v\n", err)
		os.Exit(1)
	}

	// Step 1: Initialize LevelDB storage
	fmt.Println("📦 Step 1: Initializing LevelDB storage...")
	start := time.Now()

	store, err := storage.NewLevelDBStore(dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open LevelDB: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	proofCache := storage.NewProofCache(store)
	stateTrie := storage.NewStateTrie(store)

	fmt.Printf("   ✓ LevelDB initialized at %s (%v)\n", dbPath, time.Since(start))
	fmt.Println()

	// Step 2: Create Merkle tree and add members
	fmt.Println("🌳 Step 2: Creating Merkle tree with members...")
	start = time.Now()

	tree := merkle.NewMerkleTree(TreeDepth)

	members := []string{
		"alice_secret_key_2024",
		"bob_secret_key_2024",
		"charlie_secret_key_2024",
		"david_secret_key_2024",
		"eve_secret_key_2024",
	}

	memberLeaves := make(map[string]int)
	for _, member := range members {
		idx, _, err := tree.InsertSecret([]byte(member))
		if err != nil {
			fmt.Printf("❌ Failed to insert member: %v\n", err)
			os.Exit(1)
		}
		memberLeaves[member] = idx
	}

	root := tree.GetRoot()
	fmt.Printf("   ✓ Added %d members to tree (depth=%d)\n", len(members), TreeDepth)
	fmt.Printf("   ✓ Merkle root: 0x%s\n", hex.EncodeToString(root.Bytes())[:16]+"...")
	fmt.Printf("   ✓ Tree creation took %v\n", time.Since(start))
	fmt.Println()

	// Update state trie
	if err := stateTrie.UpdateRoot(root.Bytes(), tree.Size()); err != nil {
		fmt.Printf("⚠️  Failed to update state trie: %v\n", err)
	}

	// Step 3: Initialize prover (includes trusted setup)
	fmt.Println("🔐 Step 3: Initializing Groth16 prover (trusted setup)...")
	start = time.Now()

	zkProver, err := prover.NewProver(TreeDepth)
	if err != nil {
		fmt.Printf("❌ Failed to initialize prover: %v\n", err)
		os.Exit(1)
	}

	setupTime := time.Since(start)
	fmt.Printf("   ✓ Circuit compiled with %d constraints\n", zkProver.ConstraintCount())
	fmt.Printf("   ✓ Trusted setup completed in %v\n", setupTime)
	fmt.Println()

	// Step 4: Generate membership proof for Alice
	fmt.Println("🔑 Step 4: Generating membership proof for Alice...")
	start = time.Now()

	aliceSecret := []byte("alice_secret_key_2024")
	aliceLeafHash := merkle.HashToBigInt(aliceSecret)
	aliceIdx := memberLeaves["alice_secret_key_2024"]

	treeProof, err := tree.GetProof(aliceIdx)
	if err != nil {
		fmt.Printf("❌ Failed to get Merkle proof: %v\n", err)
		os.Exit(1)
	}

	witness := &prover.MembershipWitness{
		Secret:       aliceSecret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     aliceLeafHash,
	}

	proofResult, err := zkProver.Prove(witness)
	if err != nil {
		fmt.Printf("❌ Proof generation failed: %v\n", err)
		os.Exit(1)
	}

	proveTime := proofResult.Duration
	fmt.Printf("   ✓ Proof generated in %v\n", proveTime)
	fmt.Printf("   ✓ Proof size: %d bytes\n", len(proofResult.ProofBytes))
	fmt.Println()

	// Step 5: Verify proof locally
	fmt.Println("✅ Step 5: Verifying proof locally...")
	start = time.Now()

	valid, err := zkProver.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
	if err != nil {
		fmt.Printf("❌ Verification error: %v\n", err)
		os.Exit(1)
	}

	verifyTime := time.Since(start)
	if valid {
		fmt.Printf("   ✓ Proof is VALID! Verified in %v\n", verifyTime)
	} else {
		fmt.Printf("   ✗ Proof is INVALID!\n")
		os.Exit(1)
	}
	fmt.Println()

	// Step 6: Store proof in LevelDB
	fmt.Println("💾 Step 6: Storing proof in LevelDB...")
	start = time.Now()

	proofData := proofResult.ToProofData()
	proofData.TreeDepth = TreeDepth

	proofID := fmt.Sprintf("alice_membership_%d", time.Now().Unix())
	if err := proofCache.StoreProof(proofID, &storage.StoredProof{
		ID:          proofID,
		Proof:       proofData.Proof,
		Root:        proofData.Root,
		LeafHash:    proofData.LeafHash,
		CreatedAt:   proofData.CreatedAt,
		TreeDepth:   proofData.TreeDepth,
		ProofTimeMs: proofData.ProofTimeMs,
		Verified:    valid,
	}); err != nil {
		fmt.Printf("❌ Failed to store proof: %v\n", err)
	} else {
		fmt.Printf("   ✓ Proof stored with ID: %s (%v)\n", proofID, time.Since(start))
	}

	// Show storage stats
	proofCount, _ := proofCache.ProofCount()
	rootCount, _ := stateTrie.RootCount()
	fmt.Printf("   ✓ Total proofs stored: %d\n", proofCount)
	fmt.Printf("   ✓ State root history: %d entries\n", rootCount)
	fmt.Println()

	// Step 7: Katana integration
	fmt.Println("🔗 Step 7: Connecting to Katana...")

	katanaURL := os.Getenv("KATANA_RPC_URL")
	if katanaURL == "" {
		katanaURL = "http://localhost:5050"
	}

	katanaClient, err := katana.NewClientWithURL(katanaURL)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to create Katana client: %v\n", err)
		fmt.Println("   ℹ️  Using mock client for demonstration")
		demonstrateWithMock(ctx, proofResult, root, aliceLeafHash)
	} else {
		// Try to connect to Katana
		if err := katanaClient.Ping(ctx); err != nil {
			fmt.Printf("   ⚠️  Katana not available at %s: %v\n", katanaURL, err)
			fmt.Println("   ℹ️  Using mock client for demonstration")
			demonstrateWithMock(ctx, proofResult, root, aliceLeafHash)
		} else {
			demonstrateWithKatana(ctx, katanaClient, proofResult, root, aliceLeafHash)
		}
	}

	// Final summary
	totalTime := time.Since(totalStart)
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        SUMMARY                                 ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Setup time:          %12v                            ║\n", setupTime.Round(time.Millisecond))
	fmt.Printf("║  Proof generation:    %12v                            ║\n", proveTime.Round(time.Millisecond))
	fmt.Printf("║  Verification:        %12v                            ║\n", verifyTime.Round(time.Millisecond))
	fmt.Printf("║  Total round-trip:    %12v                            ║\n", (proveTime + verifyTime).Round(time.Millisecond))
	fmt.Printf("║  Total demo time:     %12v                            ║\n", totalTime.Round(time.Millisecond))
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	roundTrip := proveTime + verifyTime
	if roundTrip < 2*time.Second {
		fmt.Println("║  🎉 Sub-2-second round-trip achieved!                          ║")
	} else {
		fmt.Println("║  ⚠️  Round-trip exceeded 2 seconds                             ║")
	}
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

func demonstrateWithMock(ctx context.Context, proofResult *prover.ProofResult, root, leafHash interface{}) {
	mock := katana.NewMockClient()

	blockNum, _ := mock.GetBlockNumber(ctx)
	chainID, _ := mock.GetChainID(ctx)
	fmt.Printf("   ✓ Mock Katana: chain=%s, block=%d\n", chainID, blockNum)

	// Simulate verification
	result, _ := mock.SimulateVerification(ctx, &katana.VerificationRequest{
		Proof: katana.Groth16Proof{
			A: katana.G1Point{X: proofResult.PublicInputs.Root, Y: proofResult.PublicInputs.LeafHash},
		},
		MerkleRoot: proofResult.PublicInputs.Root,
		LeafHash:   proofResult.PublicInputs.LeafHash,
	})

	if result.Valid {
		fmt.Printf("   ✓ Mock on-chain verification: VALID (gas: %d)\n", result.GasUsed)
	} else {
		fmt.Printf("   ✗ Mock on-chain verification: INVALID\n")
	}
}

func demonstrateWithKatana(ctx context.Context, client *katana.Client, proofResult *prover.ProofResult, root, leafHash interface{}) {
	status, err := client.GetChainStatus(ctx)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to get chain status: %v\n", err)
		return
	}

	fmt.Printf("   ✓ Connected to Katana: chain=%s, block=%d\n", status.ChainID, status.BlockNumber)

	// Note: Full on-chain verification would require deployed contracts
	fmt.Println("   ℹ️  On-chain verification requires deployed verifier contracts")
	fmt.Println("   ℹ️  Run 'make deploy' to deploy contracts and enable full verification")
}
