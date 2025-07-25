# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o nectar .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/nectar .

# Copy migrations
COPY --from=builder /build/migrations ./migrations

# Copy web templates for dashboard
COPY --from=builder /build/web ./web

# Create logs directory
RUN mkdir -p logs

# Set environment defaults
ENV CARDANO_NETWORK_MAGIC=764824073
ENV DB_CONNECTION_POOL=8
ENV WORKER_COUNT=8

# Expose metrics port and web dashboard port
EXPOSE 9090 8080

# Run the indexer
CMD ["./nectar"]