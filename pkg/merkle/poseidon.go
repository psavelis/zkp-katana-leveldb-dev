// Package merkle provides Poseidon-optimized Merkle tree implementation
// compatible with gnark circuits for ZK proof generation.
package merkle

import (
	"hash"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
)

// PoseidonHash computes a SNARK-friendly hash using MiMC (as Poseidon substitute).
// Note: gnark-crypto provides MiMC which is well-suited for SNARKs on BN254.
// For true Poseidon, additional implementation would be needed, but MiMC
// provides similar SNARK efficiency and is natively supported by gnark.
type PoseidonHash struct {
	hasher hash.Hash
}

// NewPoseidonHash creates a new Poseidon-compatible hasher.
// Uses MiMC under the hood for gnark compatibility.
func NewPoseidonHash() *PoseidonHash {
	return &PoseidonHash{
		hasher: mimc.NewMiMC(),
	}
}

// Hash computes the hash of the given data.
func (p *PoseidonHash) Hash(data ...[]byte) []byte {
	p.hasher.Reset()
	for _, d := range data {
		p.hasher.Write(d)
	}
	return p.hasher.Sum(nil)
}

// HashPair computes the hash of two elements (used for Merkle tree).
func (p *PoseidonHash) HashPair(left, right []byte) []byte {
	p.hasher.Reset()
	p.hasher.Write(left)
	p.hasher.Write(right)
	return p.hasher.Sum(nil)
}

// HashElements computes the hash of field elements.
func (p *PoseidonHash) HashElements(elements ...*big.Int) *big.Int {
	p.hasher.Reset()
	for _, e := range elements {
		var elem fr.Element
		elem.SetBigInt(e)
		bytes := elem.Bytes()
		p.hasher.Write(bytes[:])
	}
	result := p.hasher.Sum(nil)
	return new(big.Int).SetBytes(result)
}

// HashToField hashes data and returns a field element.
func (p *PoseidonHash) HashToField(data []byte) *big.Int {
	p.hasher.Reset()
	p.hasher.Write(data)
	result := p.hasher.Sum(nil)
	return new(big.Int).SetBytes(result)
}

// HashBytes is a convenience function for hashing arbitrary bytes.
func HashBytes(data ...[]byte) []byte {
	h := NewPoseidonHash()
	return h.Hash(data...)
}

// HashPairBytes is a convenience function for hashing a pair of byte slices.
func HashPairBytes(left, right []byte) []byte {
	h := NewPoseidonHash()
	return h.HashPair(left, right)
}

// HashToBigInt hashes data and returns a big.Int.
func HashToBigInt(data []byte) *big.Int {
	h := NewPoseidonHash()
	return h.HashToField(data)
}

// EmptyLeaf returns the hash of an empty leaf (zero value).
func EmptyLeaf() []byte {
	var zero fr.Element
	bytes := zero.Bytes()
	return HashBytes(bytes[:])
}

// EmptyLeafBigInt returns the hash of an empty leaf as big.Int.
func EmptyLeafBigInt() *big.Int {
	return new(big.Int).SetBytes(EmptyLeaf())
}
