#!/bin/bash
# Katana Setup Script
set -e

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              Katana Local Node Setup                           ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if Katana is already running
if curl -s http://localhost:5050 > /dev/null 2>&1; then
    echo -e "${GREEN}Katana is already running on port 5050${NC}"
    exit 0
fi

# Method 1: Try native katana
if command -v katana &> /dev/null; then
    echo "Starting Katana (native)..."
    katana --dev --dev.no-fee --host 0.0.0.0 &
    KATANA_PID=$!
    echo "Katana started with PID: $KATANA_PID"

# Method 2: Try Docker
elif command -v docker &> /dev/null; then
    echo "Starting Katana (Docker)..."
    docker run -d --name katana-local -p 5050:5050 \
        ghcr.io/dojoengine/katana:latest \
        --dev --dev.no-fee --host 0.0.0.0

# Method 3: Instructions for installation
else
    echo -e "${YELLOW}Neither katana nor Docker found.${NC}"
    echo
    echo "To install Katana:"
    echo
    echo "  Option 1: Install Dojo toolchain"
    echo "    curl -L https://install.dojoengine.org | bash"
    echo "    dojoup"
    echo
    echo "  Option 2: Use Docker"
    echo "    docker pull ghcr.io/dojoengine/katana:latest"
    echo
    exit 1
fi

# Wait for Katana to be ready
echo "Waiting for Katana to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:5050 > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Katana is ready on http://localhost:5050${NC}"
        echo
        echo "Chain ID: SN_KATANA"
        echo "Block time: instant"
        echo "Gas: disabled (dev mode)"
        echo
        echo "Pre-funded accounts:"
        curl -s http://localhost:5050 -X POST -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","method":"katana_predeployedAccounts","params":[],"id":1}' \
            2>/dev/null | jq -r '.result[0:3][] | "  \(.address)"' 2>/dev/null || true
        exit 0
    fi
    sleep 1
done

echo -e "${RED}Katana failed to start within 30 seconds${NC}"
exit 1
