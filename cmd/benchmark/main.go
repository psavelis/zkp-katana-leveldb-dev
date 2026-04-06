// Benchmark tool for ZK-SNARK performance testing
//
// Measures and reports performance metrics for:
// - Circuit setup
// - Proof generation
// - Proof verification
// - LevelDB operations
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/merkle"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/prover"
	"github.com/psavelis/zkp-katana-leveldb-dev/pkg/storage"
)

const (
	DefaultTreeDepth = 10 // Use 20 for production benchmarks
	NumIterations    = 3
	WarmupRuns       = 1
)

type BenchmarkResult struct {
	Name     string
	Duration time.Duration
	Ops      int
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ZK-SNARK Performance Benchmark                         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	treeDepth := DefaultTreeDepth
	if len(os.Args) > 1 && os.Args[1] == "--full" {
		treeDepth = 20
		fmt.Println("Running FULL benchmark with depth=20 (production settings)")
	} else {
		fmt.Println("Running quick benchmark with depth=10 (use --full for depth=20)")
	}
	fmt.Println()

	var results []BenchmarkResult

	// Benchmark 1: Merkle Tree Operations
	fmt.Println("📊 Benchmarking Merkle Tree Operations...")
	merkleResults := benchmarkMerkle(treeDepth)
	results = append(results, merkleResults...)
	fmt.Println()

	// Benchmark 2: Prover Setup
	fmt.Println("📊 Benchmarking Prover Setup...")
	setupResult, zkProver := benchmarkSetup(treeDepth)
	results = append(results, setupResult)
	fmt.Println()

	// Benchmark 3: Proof Generation
	fmt.Println("📊 Benchmarking Proof Generation...")
	proveResults := benchmarkProve(zkProver, treeDepth)
	results = append(results, proveResults...)
	fmt.Println()

	// Benchmark 4: Proof Verification
	fmt.Println("📊 Benchmarking Proof Verification...")
	verifyResults := benchmarkVerify(zkProver, treeDepth)
	results = append(results, verifyResults...)
	fmt.Println()

	// Benchmark 5: LevelDB Operations
	fmt.Println("📊 Benchmarking LevelDB Operations...")
	storageResults := benchmarkStorage()
	results = append(results, storageResults...)
	fmt.Println()

	// Print summary
	printSummary(results, treeDepth)
}

func benchmarkMerkle(depth int) []BenchmarkResult {
	var results []BenchmarkResult

	// Tree creation and insertion
	start := time.Now()
	tree := merkle.NewMerkleTree(depth)
	for i := 0; i < 100; i++ {
		_, _, _ = tree.InsertSecret([]byte(fmt.Sprintf("secret_%d", i)))
	}
	insertTime := time.Since(start)
	results = append(results, BenchmarkResult{
		Name:     "Merkle: Insert 100 leaves",
		Duration: insertTime,
		Ops:      100,
	})
	fmt.Printf("   Insert 100 leaves: %v\n", insertTime)

	// Root computation
	start = time.Now()
	for i := 0; i < 100; i++ {
		_ = tree.GetRoot()
	}
	rootTime := time.Since(start) / 100
	results = append(results, BenchmarkResult{
		Name:     "Merkle: Get root (avg)",
		Duration: rootTime,
		Ops:      1,
	})
	fmt.Printf("   Get root (avg): %v\n", rootTime)

	// Proof generation
	start = time.Now()
	for i := 0; i < 100; i++ {
		_, _ = tree.GetProof(i % tree.Size())
	}
	proofTime := time.Since(start) / 100
	results = append(results, BenchmarkResult{
		Name:     "Merkle: Generate proof (avg)",
		Duration: proofTime,
		Ops:      1,
	})
	fmt.Printf("   Generate proof (avg): %v\n", proofTime)

	return results
}

func benchmarkSetup(depth int) (BenchmarkResult, *prover.Prover) {
	// Warmup
	for i := 0; i < WarmupRuns; i++ {
		_, _ = prover.NewProver(depth)
	}

	start := time.Now()
	zkProver, err := prover.NewProver(depth)
	setupTime := time.Since(start)

	if err != nil {
		fmt.Printf("   ❌ Setup failed: %v\n", err)
		return BenchmarkResult{Name: "Prover: Setup", Duration: 0}, nil
	}

	fmt.Printf("   Setup time: %v\n", setupTime)
	fmt.Printf("   Constraints: %d\n", zkProver.ConstraintCount())

	return BenchmarkResult{
		Name:     "Prover: Setup",
		Duration: setupTime,
		Ops:      1,
	}, zkProver
}

func benchmarkProve(zkProver *prover.Prover, depth int) []BenchmarkResult {
	if zkProver == nil {
		return nil
	}

	var results []BenchmarkResult

	// Create test data
	tree := merkle.NewMerkleTree(depth)
	secret := []byte("benchmark_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &prover.MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	// Warmup
	for i := 0; i < WarmupRuns; i++ {
		_, _ = zkProver.Prove(witness)
	}

	// Benchmark
	var totalTime time.Duration
	var proofBytes []byte

	for i := 0; i < NumIterations; i++ {
		start := time.Now()
		result, err := zkProver.Prove(witness)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("   ❌ Prove failed: %v\n", err)
			continue
		}

		totalTime += elapsed
		proofBytes = result.ProofBytes
		fmt.Printf("   Run %d: %v\n", i+1, elapsed)
	}

	avgTime := totalTime / time.Duration(NumIterations)
	results = append(results, BenchmarkResult{
		Name:     "Prover: Generate proof (avg)",
		Duration: avgTime,
		Ops:      1,
	})

	fmt.Printf("   Average: %v\n", avgTime)
	fmt.Printf("   Proof size: %d bytes\n", len(proofBytes))

	return results
}

func benchmarkVerify(zkProver *prover.Prover, depth int) []BenchmarkResult {
	if zkProver == nil {
		return nil
	}

	var results []BenchmarkResult

	// Create and generate a proof first
	tree := merkle.NewMerkleTree(depth)
	secret := []byte("benchmark_secret")
	_, leafHash, _ := tree.InsertSecret(secret)
	treeProof, _ := tree.GetProof(0)
	root := tree.GetRoot()

	witness := &prover.MembershipWitness{
		Secret:       secret,
		PathElements: treeProof.PathElements,
		PathIndices:  treeProof.PathIndices,
		Root:         root,
		LeafHash:     leafHash,
	}

	proofResult, _ := zkProver.Prove(witness)

	// Warmup
	for i := 0; i < WarmupRuns; i++ {
		_, _ = zkProver.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
	}

	// Benchmark
	var totalTime time.Duration
	for i := 0; i < NumIterations*10; i++ {
		start := time.Now()
		valid, err := zkProver.Verify(proofResult.ProofBytes, proofResult.PublicInputs)
		elapsed := time.Since(start)

		if err != nil || !valid {
			fmt.Printf("   ❌ Verify failed\n")
			continue
		}

		totalTime += elapsed
	}

	avgTime := totalTime / time.Duration(NumIterations*10)
	results = append(results, BenchmarkResult{
		Name:     "Prover: Verify proof (avg)",
		Duration: avgTime,
		Ops:      1,
	})

	fmt.Printf("   Average: %v\n", avgTime)

	return results
}

func benchmarkStorage() []BenchmarkResult {
	var results []BenchmarkResult

	dbPath := filepath.Join(os.TempDir(), "benchmark.db")
	defer os.RemoveAll(dbPath)

	store, err := storage.NewLevelDBStore(dbPath)
	if err != nil {
		fmt.Printf("   ❌ Failed to open LevelDB: %v\n", err)
		return nil
	}
	defer store.Close()

	// Write benchmark
	start := time.Now()
	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		value := []byte(hex.EncodeToString(make([]byte, 256)))
		_ = store.Put(key, value)
	}
	writeTime := time.Since(start)
	avgWrite := writeTime / 1000

	results = append(results, BenchmarkResult{
		Name:     "LevelDB: Write (avg)",
		Duration: avgWrite,
		Ops:      1,
	})
	fmt.Printf("   Write 1000 entries: %v (avg: %v)\n", writeTime, avgWrite)

	// Read benchmark
	start = time.Now()
	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		_, _ = store.Get(key)
	}
	readTime := time.Since(start)
	avgRead := readTime / 1000

	results = append(results, BenchmarkResult{
		Name:     "LevelDB: Read (avg)",
		Duration: avgRead,
		Ops:      1,
	})
	fmt.Printf("   Read 1000 entries: %v (avg: %v)\n", readTime, avgRead)

	// Batch write benchmark
	start = time.Now()
	ops := make([]storage.BatchOp, 100)
	for i := 0; i < 100; i++ {
		ops[i] = storage.BatchOp{
			Key:   []byte(fmt.Sprintf("batch_%d", i)),
			Value: []byte(hex.EncodeToString(make([]byte, 256))),
		}
	}
	_ = store.BatchWrite(ops)
	batchTime := time.Since(start)

	results = append(results, BenchmarkResult{
		Name:     "LevelDB: Batch write 100",
		Duration: batchTime,
		Ops:      100,
	})
	fmt.Printf("   Batch write 100: %v\n", batchTime)

	return results
}

func printSummary(results []BenchmarkResult, depth int) {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     BENCHMARK SUMMARY                          ║")
	fmt.Printf("║                     Tree Depth: %d                              ║\n", depth)
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	var proveTime, verifyTime time.Duration

	for _, r := range results {
		status := "✓"
		if r.Duration > 2*time.Second && (r.Name == "Prover: Generate proof (avg)" || r.Name == "Prover: Verify proof (avg)") {
			status = "⚠️"
		}

		fmt.Printf("║ %s %-35s %12v ║\n", status, r.Name, r.Duration.Round(time.Microsecond))

		if r.Name == "Prover: Generate proof (avg)" {
			proveTime = r.Duration
		}
		if r.Name == "Prover: Verify proof (avg)" {
			verifyTime = r.Duration
		}
	}

	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	totalRoundTrip := proveTime + verifyTime
	fmt.Printf("║ 🔄 Total Round-trip:               %12v               ║\n", totalRoundTrip.Round(time.Microsecond))

	if totalRoundTrip < 2*time.Second {
		fmt.Println("║ 🎉 TARGET MET: Sub-2-second round-trip!                        ║")
	} else {
		fmt.Println("║ ⚠️  TARGET MISSED: Round-trip exceeds 2 seconds                ║")
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	// Exit with error code if target missed (for CI)
	if totalRoundTrip >= 2*time.Second {
		fmt.Println("\nNote: Use smaller tree depth or optimize for faster performance")
	}
}
