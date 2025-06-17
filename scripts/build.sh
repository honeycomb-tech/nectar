#!/bin/bash
# Nectar Build Script
# Builds the indexer with version information and optimizations

set -e

# Get version from git
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "Building Nectar v${VERSION}..."

# Build with version information and optimizations
go build -ldflags="-s -w \
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.GitCommit=${COMMIT}" \
    -o nectar \
    .

# Verify build
if [ -f ./nectar ]; then
    echo "Build successful!"
    echo "Binary size: $(du -h nectar | cut -f1)"
    echo ""
    echo "Version: ${VERSION}"
    echo "Commit: ${COMMIT}"
    echo "Built: ${BUILD_TIME}"
else
    echo "Build failed!"
    exit 1
fi