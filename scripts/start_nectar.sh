#!/bin/bash
# Nectar Production Startup Script
# Validates environment and starts the indexer with proper configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Nectar Cardano Indexer...${NC}"

# Check required environment variables
if [ -z "$TIDB_DSN" ]; then
    echo -e "${RED}ERROR: TIDB_DSN environment variable is not set${NC}"
    echo "Please set TIDB_DSN to your TiDB connection string"
    echo "Example: export TIDB_DSN='user:password@tcp(host:port)/nectar?charset=utf8mb4&parseTime=True&loc=Local'"
    exit 1
fi

# Check if Cardano node socket exists
SOCKET_PATH="${CARDANO_NODE_SOCKET:-/opt/cardano/cnode/sockets/node.socket}"
if [ ! -S "$SOCKET_PATH" ]; then
    echo -e "${YELLOW}WARNING: Cardano node socket not found at $SOCKET_PATH${NC}"
    echo "Make sure your Cardano node is running or set CARDANO_NODE_SOCKET"
fi

# Display configuration
echo -e "${GREEN}Configuration:${NC}"
echo "  Database: ${TIDB_DSN%@*}@***" # Hide password
echo "  Network Magic: ${CARDANO_NETWORK_MAGIC:-764824073}"
echo "  Socket Path: $SOCKET_PATH"
echo "  Workers: ${WORKER_COUNT:-8}"
echo "  DB Pool Size: ${DB_CONNECTION_POOL:-8}"
echo "  Bulk Mode: ${BULK_MODE_ENABLED:-false}"

# Optional: Load from .env file if it exists
if [ -f .env ]; then
    echo -e "${YELLOW}Loading configuration from .env file...${NC}"
    export $(grep -v '^#' .env | xargs)
fi

# Build the binary if it doesn't exist
if [ ! -f ./nectar ]; then
    echo -e "${YELLOW}Building Nectar...${NC}"
    go build -o nectar .
fi

# Create required directories
mkdir -p logs

# Start Nectar
echo -e "${GREEN}Starting indexer...${NC}"
exec ./nectar 2>&1 | tee -a logs/nectar.log