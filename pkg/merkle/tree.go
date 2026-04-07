package merkle

import (
	"errors"
	"fmt"
	"math/big"
)

// TreeDepth is the default depth for Merkle trees (supports ~1M leaves).
const TreeDepth = 20

// MerkleProof contains the proof elements for membership verification.
type MerkleProof struct {
	// PathElements contains the sibling hashes from leaf to root.
	PathElements []*big.Int
	// PathIndices indicates the position at each level (0 = left, 1 = right).
	PathIndices []int
	// LeafIndex is the index of the leaf in the tree.
	LeafIndex int
}

// MerkleTree is a sparse Merkle tree implementation using Poseidon hash.
type MerkleTree struct {
	depth    int
	leaves   map[int]*big.Int // Sparse storage: index -> leaf hash
	nextLeaf int              // Next available leaf index
	hasher   *PoseidonHash
	zeros    []*big.Int // Pre-computed empty subtree hashes
}

// NewMerkleTree creates a new Merkle tree with the specified depth.
func NewMerkleTree(depth int) *MerkleTree {
	if depth <= 0 {
		depth = TreeDepth
	}

	t := &MerkleTree{
		depth:    depth,
		leaves:   make(map[int]*big.Int),
		nextLeaf: 0,
		hasher:   NewPoseidonHash(),
		zeros:    make([]*big.Int, depth+1),
	}

	// Pre-compute empty subtree hashes (zeros).
	// zeros[0] = hash of empty leaf
	// zeros[i] = hash(zeros[i-1], zeros[i-1])
	t.zeros[0] = EmptyLeafBigInt()
	for i := 1; i <= depth; i++ {
		t.zeros[i] = t.hasher.HashElements(t.zeros[i-1], t.zeros[i-1])
	}

	return t
}

// Insert adds a new leaf to the tree and returns its index.
func (t *MerkleTree) Insert(leafHash *big.Int) (int, error) {
	maxLeaves := 1 << t.depth
	if t.nextLeaf >= maxLeaves {
		return -1, errors.New("tree is full")
	}

	index := t.nextLeaf
	t.leaves[index] = new(big.Int).Set(leafHash)
	t.nextLeaf++

	return index, nil
}

// InsertBytes adds a new leaf from bytes and returns its index.
func (t *MerkleTree) InsertBytes(data []byte) (int, error) {
	leafHash := HashToBigInt(data)
	return t.Insert(leafHash)
}

// InsertSecret hashes a secret and inserts it as a leaf.
func (t *MerkleTree) InsertSecret(secret []byte) (int, *big.Int, error) {
	leafHash := HashToBigInt(secret)
	index, err := t.Insert(leafHash)
	return index, leafHash, err
}

// GetLeaf returns the leaf hash at the given index.
func (t *MerkleTree) GetLeaf(index int) (*big.Int, error) {
	if index < 0 || index >= t.nextLeaf {
		return nil, fmt.Errorf("leaf index %d out of range [0, %d)", index, t.nextLeaf)
	}
	leaf, exists := t.leaves[index]
	if !exists {
		return t.zeros[0], nil // Return empty leaf hash
	}
	return leaf, nil
}

// GetRoot computes and returns the current Merkle root.
func (t *MerkleTree) GetRoot() *big.Int {
	if t.nextLeaf == 0 {
		// Empty tree - return root of all zeros
		return t.zeros[t.depth]
	}
	return t.computeRoot()
}

// GetRootBytes returns the root as a byte slice.
func (t *MerkleTree) GetRootBytes() []byte {
	root := t.GetRoot()
	return root.Bytes()
}

// computeRoot calculates the Merkle root from current leaves.
func (t *MerkleTree) computeRoot() *big.Int {
	// Build the tree bottom-up
	currentLevel := make(map[int]*big.Int)

	// Copy leaves to current level
	for k, v := range t.leaves {
		currentLevel[k] = v
	}

	// Process each level from bottom to top
	for level := 0; level < t.depth; level++ {
		nextLevel := make(map[int]*big.Int)
		processed := make(map[int]bool)

		for idx := range currentLevel {
			parentIdx := idx / 2
			if processed[parentIdx] {
				continue
			}
			processed[parentIdx] = true

			leftIdx := parentIdx * 2
			rightIdx := parentIdx*2 + 1

			left := t.getNodeOrZero(currentLevel, leftIdx, level)
			right := t.getNodeOrZero(currentLevel, rightIdx, level)

			nextLevel[parentIdx] = t.hasher.HashElements(left, right)
		}

		currentLevel = nextLevel
	}

	// Root should be at index 0
	if root, exists := currentLevel[0]; exists {
		return root
	}
	return t.zeros[t.depth]
}

// getNodeOrZero returns the node value or the appropriate zero value.
func (t *MerkleTree) getNodeOrZero(level map[int]*big.Int, idx int, levelNum int) *big.Int {
	if val, exists := level[idx]; exists {
		return val
	}
	return t.zeros[levelNum]
}

// GetProof generates a Merkle proof for the leaf at the given index.
func (t *MerkleTree) GetProof(index int) (*MerkleProof, error) {
	if index < 0 || index >= t.nextLeaf {
		return nil, fmt.Errorf("leaf index %d out of range [0, %d)", index, t.nextLeaf)
	}

	proof := &MerkleProof{
		PathElements: make([]*big.Int, t.depth),
		PathIndices:  make([]int, t.depth),
		LeafIndex:    index,
	}

	// Build all tree levels first
	levels := t.buildLevels()

	// Extract proof path
	currentIdx := index
	for level := 0; level < t.depth; level++ {
		siblingIdx := currentIdx ^ 1 // XOR to get sibling index
		proof.PathIndices[level] = currentIdx & 1

		// Get sibling from computed levels
		if sibling, exists := levels[level][siblingIdx]; exists {
			proof.PathElements[level] = sibling
		} else {
			proof.PathElements[level] = t.zeros[level]
		}

		currentIdx = currentIdx / 2
	}

	return proof, nil
}

// buildLevels constructs all levels of the tree.
func (t *MerkleTree) buildLevels() []map[int]*big.Int {
	levels := make([]map[int]*big.Int, t.depth+1)

	// Level 0 is the leaves
	levels[0] = make(map[int]*big.Int)
	for k, v := range t.leaves {
		levels[0][k] = v
	}

	// Build each subsequent level
	for level := 1; level <= t.depth; level++ {
		levels[level] = make(map[int]*big.Int)
		prevLevel := levels[level-1]

		processed := make(map[int]bool)
		for idx := range prevLevel {
			parentIdx := idx / 2
			if processed[parentIdx] {
				continue
			}
			processed[parentIdx] = true

			leftIdx := parentIdx * 2
			rightIdx := parentIdx*2 + 1

			left := t.getNodeOrZero(prevLevel, leftIdx, level-1)
			right := t.getNodeOrZero(prevLevel, rightIdx, level-1)

			levels[level][parentIdx] = t.hasher.HashElements(left, right)
		}
	}

	return levels
}

// VerifyProof verifies a Merkle proof against a root.
func (t *MerkleTree) VerifyProof(leafHash, root *big.Int, proof *MerkleProof) bool {
	return VerifyMerkleProof(leafHash, root, proof)
}

// VerifyMerkleProof is a standalone proof verification function.
func VerifyMerkleProof(leafHash, root *big.Int, proof *MerkleProof) bool {
	hasher := NewPoseidonHash()
	current := new(big.Int).Set(leafHash)

	for i := 0; i < len(proof.PathElements); i++ {
		sibling := proof.PathElements[i]
		if proof.PathIndices[i] == 0 {
			// Current node is on the left
			current = hasher.HashElements(current, sibling)
		} else {
			// Current node is on the right
			current = hasher.HashElements(sibling, current)
		}
	}

	return current.Cmp(root) == 0
}

// Depth returns the tree depth.
func (t *MerkleTree) Depth() int {
	return t.depth
}

// Size returns the number of inserted leaves.
func (t *MerkleTree) Size() int {
	return t.nextLeaf
}

// Capacity returns the maximum number of leaves.
func (t *MerkleTree) Capacity() int {
	return 1 << t.depth
}

// GetZeros returns the pre-computed zero hashes.
func (t *MerkleTree) GetZeros() []*big.Int {
	zeros := make([]*big.Int, len(t.zeros))
	for i, z := range t.zeros {
		zeros[i] = new(big.Int).Set(z)
	}
	return zeros
}
