#!/bin/bash

# Nectar Startup Script
# This script starts the Nectar Cardano blockchain indexer with proper environment setup

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}====================================${NC}"
echo -e "${BLUE}     NECTAR BLOCKCHAIN INDEXER      ${NC}"
echo -e "${BLUE}====================================${NC}"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found!${NC}"
    echo "Creating .env from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}Created .env file. Please update it with your database credentials.${NC}"
        exit 1
    else
        echo -e "${RED}Error: .env.example not found either!${NC}"
        exit 1
    fi
fi

# Load environment variables
while IFS='=' read -r key value; do
    # Skip comments and empty lines
    [[ $key =~ ^#.*$ ]] && continue
    [[ -z $key ]] && continue
    # Remove quotes if present and export
    value="${value%\"}"
    value="${value#\"}"
    export "$key=$value"
done < .env

# Verify required environment variables
if [ -z "$TIDB_DSN" ]; then
    echo -e "${RED}Error: TIDB_DSN not set in .env file${NC}"
    exit 1
fi

if [ -z "$CARDANO_NODE_SOCKET" ]; then
    echo -e "${RED}Error: CARDANO_NODE_SOCKET not set in .env file${NC}"
    exit 1
fi

# Check if socket exists
if [ ! -S "$CARDANO_NODE_SOCKET" ]; then
    echo -e "${RED}Warning: Cardano node socket not found at $CARDANO_NODE_SOCKET${NC}"
    echo "Make sure the Cardano node is running"
fi

# Display current configuration
echo -e "${GREEN}Configuration:${NC}"
echo "  Database: TiDB"
echo "  Socket: $CARDANO_NODE_SOCKET"
echo "  Network: Mainnet (magic: $CARDANO_NETWORK_MAGIC)"
echo "  Dashboard: $DASHBOARD_TYPE"
if [[ "$DASHBOARD_TYPE" == "web" || "$DASHBOARD_TYPE" == "both" ]]; then
    echo "  Web Port: ${WEB_PORT:-8080}"
fi
echo ""

# Build Nectar if binary doesn't exist
if [ ! -f ./nectar ]; then
    echo -e "${BLUE}Building Nectar...${NC}"
    go build -o nectar .
    if [ $? -ne 0 ]; then
        echo -e "${RED}Build failed!${NC}"
        exit 1
    fi
    echo -e "${GREEN}Build successful!${NC}"
fi

# Start Nectar
echo -e "${BLUE}Starting Nectar...${NC}"
if [[ "$DASHBOARD_TYPE" == "web" || "$DASHBOARD_TYPE" == "both" ]]; then
    echo -e "${GREEN}Web dashboard will be available at http://localhost:${WEB_PORT:-8080}${NC}"
fi
echo ""

# Run Nectar
exec ./nectar