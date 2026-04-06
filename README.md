# ZK-SNARK Development Environment

A complete ZK-SNARK development environment using **gnark**, **LevelDB**, and **Katana** for blazing-fast local iteration.

[![CI](https://github.com/psavelis/zkp-katana-leveldb-dev/actions/workflows/ci.yml/badge.svg)](https://github.com/psavelis/zkp-katana-leveldb-dev/actions/workflows/ci.yml)

## 60-Second Quickstart

```bash
# Clone the repository
git clone https://github.com/psavelis/zkp-katana-leveldb-dev.git
cd zkp-katana-leveldb-dev

# Option 1: Docker (recommended)
docker-compose -f docker/docker-compose.yml up

# Option 2: Local development
go mod download
go run ./cmd/demo
```

## Features

- **gnark-powered ZK circuits**: Membership proof circuit with Merkle tree verification
- **Poseidon-optimized Merkle trees**: SNARK-efficient hashing with depth-20 support (~1M leaves)
- **Groth16 on BN254**: Fast proof generation with Ethereum-compatible curve
- **LevelDB persistence**: Production-like storage for proofs and state
- **Katana integration**: Local Starknet sequencer for on-chain verification
- **Sub-second iteration**: Full proof cycle under 2 seconds

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           User Secret                               │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Poseidon Hash (MiMC)                           │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Merkle Tree (Depth 20)                           │
│              ┌──────────────────────────────────┐                   │
│              │           Root                   │                   │
│              └──────────────┬───────────────────┘                   │
│                    ┌────────┴────────┐                              │
│                    ▼                 ▼                              │
│                 [...]             [...]                             │
│                    │                 │                              │
│              ┌─────┴─────┐     ┌─────┴─────┐                        │
│              ▼           ▼     ▼           ▼                        │
│           Leaf₀       Leaf₁  Leaf₂      Leaf₃                       │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   MembershipCircuit (gnark)                         │
│                                                                     │
│  Public Inputs:  Root, LeafHash                                     │
│  Private Inputs: Secret, PathElements, PathIndices                  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Groth16 Prover (BN254)                          │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
            ┌──────────────────┼──────────────────┐
            ▼                  ▼                  ▼
┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐
│     LevelDB       │ │  Local Verifier   │ │     Katana        │
│   Proof Cache     │ │   (Groth16)       │ │  Cairo Contract   │
└───────────────────┘ └───────────────────┘ └───────────────────┘
```

## Project Structure

```
zkp-katana-leveldb-dev/
├── cmd/
│   ├── demo/          # Demo application
│   └── benchmark/     # Performance benchmarks
├── pkg/
│   ├── circuit/       # gnark circuit definitions
│   ├── prover/        # Groth16 prover/verifier
│   ├── merkle/        # Poseidon Merkle tree
│   ├── storage/       # LevelDB storage layer
│   ├── katana/        # Starknet RPC client
│   └── e2e/           # Integration tests
├── contracts/         # Cairo verifier contracts
├── scripts/           # Utility scripts
├── docker/            # Docker configuration
└── .github/workflows/ # CI/CD
```

## Usage

### Run Demo

```bash
# Build and run
make demo

# Or directly
go run ./cmd/demo
```

### Run Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# E2E tests (requires Katana)
make e2e
```

### Run Benchmarks

```bash
# Quick benchmark (depth=10)
make benchmark

# Full benchmark (depth=20)
go run ./cmd/benchmark --full
```

### Docker

```bash
# Start all services
make docker-up

# Stop
make docker-down

# Build only
docker-compose -f docker/docker-compose.yml build
```

### Cairo Contracts

```bash
# Build contracts
make contracts

# Deploy to Katana
./scripts/deploy-contracts.sh
```

## Performance Targets

| Operation | Target | Typical |
|-----------|--------|---------|
| Proof Generation | <1.5s | ~800ms |
| Verification | <100ms | ~50ms |
| LevelDB Write | <10ms | ~1ms |
| **Full Round-trip** | **<2s** | **~900ms** |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LEVELDB_PATH` | `./data/demo.db` | LevelDB database path |
| `KATANA_RPC_URL` | `http://localhost:5050` | Katana RPC endpoint |

## Dependencies

- Go 1.22+
- [gnark](https://github.com/Consensys/gnark) - ZK-SNARK library
- [goleveldb](https://github.com/syndtr/goleveldb) - LevelDB Go implementation
- [Katana](https://github.com/dojoengine/dojo) - Starknet local sequencer

### Optional

- [Scarb](https://docs.swmansion.com/scarb/) - Cairo package manager (for contracts)
- [Starkli](https://book.starkli.rs/) - Starknet CLI (for deployment)
- Docker & Docker Compose

## How It Works

1. **Merkle Tree**: Members are hashed using Poseidon (MiMC) and inserted into a sparse Merkle tree
2. **Circuit**: The `MembershipCircuit` proves knowledge of a secret that hashes to a leaf in the tree
3. **Proof**: Groth16 generates a succinct proof (~200 bytes) that can be verified efficiently
4. **Storage**: Proofs and state roots are persisted in LevelDB
5. **On-chain**: Optional verification via Cairo contracts on Katana

## Use Cases

- **Privacy-preserving membership**: Prove you belong to a group without revealing identity
- **Compliance**: Selective disclosure for regulatory requirements
- **Rollups**: Off-chain proof generation with on-chain verification
- **Airdrops**: Prove eligibility without revealing wallet address

## Development

```bash
# Setup
go mod download

# Format
make fmt

# Lint
make lint

# Clean
make clean
```

## Contributing

Contributions welcome! Please read the contributing guidelines first.

## License

MIT

---

Built with gnark, LevelDB, and Katana for blazing-fast ZK development.
