#!/bin/bash
# Benchmark Script
set -e

echo "Running ZKP Benchmark..."
echo

cd "$(dirname "$0")/.."

# Parse arguments
FULL=false
if [[ "$1" == "--full" ]]; then
    FULL=true
fi

if [ "$FULL" = true ]; then
    echo "Running FULL benchmark (depth=20)..."
    go run ./cmd/benchmark --full
else
    echo "Running quick benchmark (depth=10)..."
    echo "Use --full for production-level benchmark"
    echo
    go run ./cmd/benchmark
fi
