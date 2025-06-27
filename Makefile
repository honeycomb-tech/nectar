# Nectar Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=nectar

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: test build

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run specific test
test-unit:
	$(GOTEST) -v -short ./...

# Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run the application
run: build
	./$(BINARY_NAME)

# Run with race detector
race:
	$(GOTEST) -race -short ./...

# Check for security vulnerabilities
security:
	$(GOCMD) list -json -m all | nancy sleuth

# Generate mocks (if using mockgen)
mocks:
	$(GOCMD) generate ./...

# Docker build
docker-build:
	docker build -t nectar:latest .

# Help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run all tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-unit     - Run unit tests only"
	@echo "  make bench         - Run benchmarks"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download dependencies"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make run           - Build and run"
	@echo "  make race          - Run tests with race detector"
	@echo "  make security      - Check for vulnerabilities"
	@echo "  make docker-build  - Build Docker image"

.PHONY: all build test test-coverage test-unit bench clean deps lint fmt run race security mocks docker-build help