#!/bin/bash

# Nectar CLI wrapper script
# This script demonstrates the new TOML configuration functionality

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build nectar if not already built
if [ ! -f "./nectar" ]; then
    echo -e "${BLUE}Building Nectar...${NC}"
    go build -o nectar .
fi

# Handle commands
case "$1" in
    init)
        echo -e "${GREEN}Initializing Nectar configuration...${NC}"
        ./nectar init "${@:2}"
        ;;
    
    migrate-env)
        echo -e "${GREEN}Migrating environment variables to TOML config...${NC}"
        ./nectar migrate-env "${@:2}"
        ;;
    
    run)
        echo -e "${GREEN}Running Nectar with configuration...${NC}"
        ./nectar --config "${2:-nectar.toml}"
        ;;
    
    version)
        ./nectar version
        ;;
    
    help)
        echo "Nectar CLI - High-performance Cardano indexer"
        echo ""
        echo "Usage:"
        echo "  ./nectar-cli.sh init [config-path]      Initialize a new configuration file"
        echo "  ./nectar-cli.sh migrate-env [config]    Migrate environment variables to config"
        echo "  ./nectar-cli.sh run [config]            Run the indexer with config file"
        echo "  ./nectar-cli.sh version                 Show version information"
        echo "  ./nectar-cli.sh help                    Show this help message"
        echo ""
        echo "Examples:"
        echo "  # Initialize default config"
        echo "  ./nectar-cli.sh init"
        echo ""
        echo "  # Initialize config at custom path"
        echo "  ./nectar-cli.sh init /etc/nectar/config.toml"
        echo ""
        echo "  # Migrate current environment to config"
        echo "  ./nectar-cli.sh migrate-env production.toml"
        echo ""
        echo "  # Run with specific config"
        echo "  ./nectar-cli.sh run production.toml"
        ;;
    
    *)
        echo -e "${YELLOW}Unknown command. Use './nectar-cli.sh help' for usage.${NC}"
        exit 1
        ;;
esac