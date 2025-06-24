#!/bin/bash

# Nectar Startup Script - TOML Configuration

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
NECTAR_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
cd "$NECTAR_ROOT"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}====================================${NC}"
echo -e "${BLUE}     NECTAR BLOCKCHAIN INDEXER      ${NC}"
echo -e "${BLUE}====================================${NC}"

# Check for nectar.toml
if [ ! -f nectar.toml ]; then
    echo -e "${RED}Error: nectar.toml not found!${NC}"
    echo "Run: ./nectar init"
    exit 1
fi

# Build if needed
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
echo -e "${GREEN}Using configuration: nectar.toml${NC}"
echo ""

exec ./nectar "$@"