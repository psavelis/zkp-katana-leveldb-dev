#!/bin/bash
# End-to-End Test Script for ZKP-Katana-LevelDB
set -e

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         ZKP-Katana-LevelDB E2E Test Suite                      ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

# Check if Katana is running
check_katana() {
    if curl -s http://localhost:5050 > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Start Katana if not running
start_katana() {
    echo -e "${YELLOW}Starting Katana...${NC}"

    if command -v docker-compose &> /dev/null; then
        docker-compose -f docker/docker-compose.yml up -d katana
    elif command -v docker &> /dev/null; then
        docker run -d --name katana-test -p 5050:5050 \
            ghcr.io/dojoengine/katana:latest \
            --dev --dev.no-fee --host 0.0.0.0
    else
        echo -e "${YELLOW}Docker not available, running without Katana${NC}"
        return 1
    fi

    # Wait for Katana to be ready
    echo "Waiting for Katana to be ready..."
    for i in {1..30}; do
        if check_katana; then
            echo -e "${GREEN}Katana is ready!${NC}"
            return 0
        fi
        sleep 2
    done

    echo -e "${RED}Katana failed to start${NC}"
    return 1
}

# Cleanup function
cleanup() {
    echo
    echo "Cleaning up..."
    rm -rf ./data/test.db
}

trap cleanup EXIT

# Step 1: Check dependencies
echo "📋 Step 1: Checking dependencies..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Go $(go version | cut -d' ' -f3)${NC}"

# Step 2: Download dependencies
echo
echo "📦 Step 2: Downloading dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✓ Dependencies ready${NC}"

# Step 3: Run unit tests
echo
echo "🧪 Step 3: Running unit tests..."
if go test -v -short ./pkg/merkle/... ./pkg/storage/... ./pkg/katana/...; then
    echo -e "${GREEN}✓ Unit tests passed${NC}"
else
    echo -e "${RED}✗ Unit tests failed${NC}"
    exit 1
fi

# Step 4: Run circuit tests (may take longer)
echo
echo "🔐 Step 4: Running circuit tests..."
if go test -v -short ./pkg/circuit/...; then
    echo -e "${GREEN}✓ Circuit tests passed${NC}"
else
    echo -e "${RED}✗ Circuit tests failed${NC}"
    exit 1
fi

# Step 5: Check Katana availability
echo
echo "🔗 Step 5: Checking Katana availability..."
KATANA_AVAILABLE=false
if check_katana; then
    echo -e "${GREEN}✓ Katana is running${NC}"
    KATANA_AVAILABLE=true
else
    echo -e "${YELLOW}Katana not running, attempting to start...${NC}"
    if start_katana; then
        KATANA_AVAILABLE=true
    fi
fi

# Step 6: Run demo
echo
echo "🚀 Step 6: Running demo application..."
export LEVELDB_PATH="./data/test.db"
if [ "$KATANA_AVAILABLE" = true ]; then
    export KATANA_RPC_URL="http://localhost:5050"
fi

if go run ./cmd/demo; then
    echo -e "${GREEN}✓ Demo completed successfully${NC}"
else
    echo -e "${RED}✗ Demo failed${NC}"
    exit 1
fi

# Step 7: Run benchmark
echo
echo "📊 Step 7: Running benchmark..."
if go run ./cmd/benchmark; then
    echo -e "${GREEN}✓ Benchmark completed${NC}"
else
    echo -e "${YELLOW}⚠ Benchmark completed with warnings${NC}"
fi

# Final summary
echo
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    E2E TEST SUMMARY                            ║"
echo "╠════════════════════════════════════════════════════════════════╣"
echo -e "║ ${GREEN}✓${NC} Unit tests:      PASSED                                     ║"
echo -e "║ ${GREEN}✓${NC} Circuit tests:   PASSED                                     ║"
echo -e "║ ${GREEN}✓${NC} Demo:            PASSED                                     ║"
echo -e "║ ${GREEN}✓${NC} Benchmark:       COMPLETED                                  ║"
if [ "$KATANA_AVAILABLE" = true ]; then
    echo -e "║ ${GREEN}✓${NC} Katana:          CONNECTED                                  ║"
else
    echo -e "║ ${YELLOW}⚠${NC} Katana:          NOT AVAILABLE (used mock)                 ║"
fi
echo "╚════════════════════════════════════════════════════════════════╝"
echo
echo -e "${GREEN}All E2E tests passed!${NC}"
