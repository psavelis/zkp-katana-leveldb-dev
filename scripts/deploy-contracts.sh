#!/bin/bash
# Deploy Cairo contracts to Katana
set -e

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║           Deploy ZKP Verifier Contracts                        ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
CONTRACTS_DIR="$PROJECT_ROOT/contracts"

# Configuration
KATANA_RPC="${KATANA_RPC_URL:-http://localhost:5050}"

# Check dependencies
echo "Checking dependencies..."

if ! command -v scarb &> /dev/null; then
    echo -e "${RED}Scarb not found. Install with:${NC}"
    echo "  curl -L https://raw.githubusercontent.com/software-mansion/scarb/main/install.sh | sh"
    exit 1
fi
echo -e "${GREEN}✓ Scarb $(scarb --version | cut -d' ' -f2)${NC}"

if ! command -v starkli &> /dev/null; then
    echo -e "${YELLOW}Starkli not found. Some features may be limited.${NC}"
    echo "  Install with: curl https://get.starkli.sh | sh"
fi

# Check Katana
echo
echo "Checking Katana at $KATANA_RPC..."
if ! curl -s "$KATANA_RPC" > /dev/null 2>&1; then
    echo -e "${RED}Katana is not running at $KATANA_RPC${NC}"
    echo "Start Katana with: ./scripts/setup-katana.sh"
    exit 1
fi
echo -e "${GREEN}✓ Katana is running${NC}"

# Build contracts
echo
echo "Building contracts..."
cd "$CONTRACTS_DIR"

if scarb build; then
    echo -e "${GREEN}✓ Contracts built successfully${NC}"
else
    echo -e "${RED}✗ Contract build failed${NC}"
    exit 1
fi

# Check build artifacts
echo
echo "Checking build artifacts..."
GROTH16_ARTIFACT="$CONTRACTS_DIR/target/dev/zkp_verifier_Groth16Verifier.contract_class.json"
MEMBERSHIP_ARTIFACT="$CONTRACTS_DIR/target/dev/zkp_verifier_MembershipVerifier.contract_class.json"

if [ -f "$GROTH16_ARTIFACT" ]; then
    echo -e "${GREEN}✓ Groth16Verifier artifact found${NC}"
else
    echo -e "${YELLOW}⚠ Groth16Verifier artifact not found${NC}"
fi

if [ -f "$MEMBERSHIP_ARTIFACT" ]; then
    echo -e "${GREEN}✓ MembershipVerifier artifact found${NC}"
else
    echo -e "${YELLOW}⚠ MembershipVerifier artifact not found${NC}"
fi

# Deploy contracts (if starkli is available)
if command -v starkli &> /dev/null; then
    echo
    echo "Deploying contracts..."

    # Get pre-funded account from Katana
    ACCOUNTS=$(curl -s "$KATANA_RPC" -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"katana_predeployedAccounts","params":[],"id":1}' \
        | jq -r '.result[0]')

    if [ -n "$ACCOUNTS" ]; then
        ACCOUNT_ADDRESS=$(echo "$ACCOUNTS" | jq -r '.address')
        PRIVATE_KEY=$(echo "$ACCOUNTS" | jq -r '.privateKey')

        echo "Using account: $ACCOUNT_ADDRESS"

        # Note: Actual deployment would require proper starkli configuration
        echo -e "${YELLOW}Note: Full deployment requires starkli account setup${NC}"
        echo "See: https://book.starkli.rs/accounts"
    fi
else
    echo
    echo -e "${YELLOW}Starkli not available for deployment${NC}"
    echo "Contracts built successfully but not deployed"
fi

echo
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    DEPLOYMENT SUMMARY                          ║"
echo "╠════════════════════════════════════════════════════════════════╣"
echo -e "║ ${GREEN}✓${NC} Contracts compiled                                          ║"
echo "║                                                                ║"
echo "║ Artifacts:                                                     ║"
echo "║   - target/dev/zkp_verifier_Groth16Verifier.contract_class.json║"
echo "║   - target/dev/zkp_verifier_MembershipVerifier.contract_class.json"
echo "╚════════════════════════════════════════════════════════════════╝"
