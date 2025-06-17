# Nectar

High-performance Cardano blockchain indexer optimized for TiDB distributed database systems.

## Overview

Nectar is a production-grade Cardano blockchain indexer designed to leverage TiDB's distributed SQL capabilities for maximum throughput. It provides comprehensive indexing of all on-chain data including transactions, stake pools, certificates, governance actions, and multi-asset information across all Cardano eras.

## Current Status

- **Processing Speed**: 800+ blocks/second (Stage 1 parallel processing implemented)
- **Era Support**: Full support for Byron through Conway eras
- **Data Completeness**: All transaction types, certificates, withdrawals, and metadata are now properly extracted

## Key Features

- **Full Era Support**: Byron, Shelley, Allegra, Mary, Alonzo, Babbage, and Conway
- **Parallel Processing**: Stage 1 implemented with concurrent transaction processing
- **Complete Data Extraction**: 
  - Stake pool registrations and delegations
  - Reward withdrawals
  - Multi-asset minting (Mary era+)
  - Smart contract data (Alonzo era+)
  - Inline datums and reference scripts (Babbage era+)
  - Governance actions (Conway era)
- **TiDB Optimizations**: Batch operations, parallel inserts, optimized indexes
- **Real-time Dashboard**: Live progress monitoring with era breakdowns
- **Checkpoint System**: Resumable synchronization with automatic recovery
- **Error Tracking**: Comprehensive error collection and reporting

## System Requirements

- Go 1.21 or higher
- TiDB 7.5+ or MySQL 8.0+
- Cardano node with N2C (node-to-client) protocol access
- Minimum 32 CPU cores (recommended)
- Minimum 64GB RAM
- 2TB+ NVMe SSD storage

## Installation

```bash
git clone https://github.com/your-repo/nectar.git
cd nectar
go build .
```

## Configuration

### Quick Start

#### Using Environment Variables

```bash
# Clone the repository
git clone https://github.com/yourusername/nectar.git
cd nectar

# Copy environment template
cp .env.example .env

# Edit .env with your configuration
# Required: Set TIDB_DSN with your database credentials

# Build and run
go build -o nectar
./nectar
```

#### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/yourusername/nectar.git
cd nectar

# Create .env file with your configuration
cp .env.example .env

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f nectar
```

#### Using Systemd (Production)

```bash
# Build the binary
go build -o nectar

# Install binary and files
sudo mkdir -p /opt/nectar /etc/nectar
sudo cp nectar /opt/nectar/
sudo cp -r migrations /opt/nectar/
sudo cp scripts/nectar.service /etc/systemd/system/
sudo cp scripts/nectar.env.example /etc/nectar/nectar.env

# Edit configuration
sudo nano /etc/nectar/nectar.env

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable nectar
sudo systemctl start nectar

# Check status
sudo systemctl status nectar
sudo journalctl -u nectar -f
```

### Database Setup

Nectar requires TiDB or MySQL-compatible database:

```bash
# Set database connection via environment variable
export TIDB_DSN="user:password@tcp(host:port)/nectar?charset=utf8mb4&parseTime=True&loc=Local"

# Database is created automatically on first run
# Migrations are applied automatically
```

### Cardano Node Connection

Nectar automatically detects Cardano node socket or uses environment variable:

```bash
# Set custom socket path (optional)
export CARDANO_NODE_SOCKET="/opt/cardano/cnode/sockets/node.socket"

# Set network magic (optional, defaults to mainnet)
export CARDANO_NETWORK_MAGIC="764824073"  # Mainnet
# export CARDANO_NETWORK_MAGIC="1"        # Preprod
# export CARDANO_NETWORK_MAGIC="2"        # Preview
```

## Usage

### Starting Fresh Sync

```bash
# Drop all tables for fresh start (optional)
mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p$DB_PASS nectar < scripts/drop_all_tables.sql

# Start indexer
./nectar
```

### Resume Existing Sync

```bash
# Nectar automatically resumes from last checkpoint
./nectar
```

### Command Line Options

```bash
./nectar [options]
  -log-level string    Set log level: debug, info, warn, error (default "info")
  -skip-migrations     Skip database migrations
  -force-resync       Force resync from genesis (drops checkpoints)
```

## Architecture

### Processing Pipeline

1. **Block Reception**: Receives blocks via ChainSync protocol
2. **Parallel Processing**: Processes transactions concurrently (32 workers)
3. **Batch Operations**: Groups database operations for efficiency
4. **Error Recovery**: Automatic retry with exponential backoff

### Core Processors

- **BlockProcessor**: Main orchestrator for block and transaction processing
- **CertificateProcessor**: Handles stake pool and delegation certificates
- **WithdrawalProcessor**: Processes reward withdrawals
- **AssetProcessor**: Multi-asset and metadata handling
- **ScriptProcessor**: Plutus script and redeemer processing
- **GovernanceProcessor**: Conway governance actions

### Database Schema

74 tables covering:
- Core blockchain data (blocks, transactions, UTXOs)
- Staking system (pools, delegations, rewards)
- Multi-assets and NFTs
- Smart contracts and scripts
- Governance proposals and votes
- Off-chain metadata

## Performance Tuning

### TiDB Optimizations

The indexer automatically applies:
- Batch insert/delete operations
- Parallel query execution
- Optimized chunk sizes
- Async commit and 1PC
- Custom indexes for common queries

### Memory Management

- Transaction semaphore limits concurrent processing
- Batch sizes optimized for memory usage
- Automatic garbage collection tuning

## Monitoring

### Built-in Dashboard

Real-time display of:
- Current era and block height
- Processing speed (blocks/sec)
- Era completion percentages
- Memory and goroutine stats
- Recent block activity
- Error summaries

### Metrics

- Block processing rate
- Transaction throughput
- Database operation latency
- Error rates by component

## Development

### Project Structure

```
nectar/
├── processors/       # Block and data processors
├── models/          # Database models
├── database/        # TiDB connection and migrations
├── dashboard/       # Terminal UI
├── connection/      # Cardano node connection
├── statequery/      # Node state queries
└── migrations/      # SQL migration files
```

### Building

```bash
go mod download
go build -o nectar .
```

### Testing

```bash
go test ./...
```

## Troubleshooting

### Common Issues

1. **"Era detection incorrect"**: Fixed - Allegra blocks now properly detected
2. **"Missing stake data"**: Fixed - All certificate processors implemented
3. **"Slow initial sync"**: Enable parallel processing (implemented in Stage 1)

### Debug Mode

```bash
# Enable debug logging
./nectar -log-level debug

# Check error logs
tail -f errors.log
tail -f unified_errors.log
```

### Database Verification

```bash
# Check table counts
mysql -h 127.0.0.1 -P 4000 -u root -p nectar -e "
SELECT 'blocks' as table_name, COUNT(*) as count FROM blocks
UNION ALL SELECT 'txes', COUNT(*) FROM txes
UNION ALL SELECT 'stake_addresses', COUNT(*) FROM stake_addresses
UNION ALL SELECT 'pool_hashes', COUNT(*) FROM pool_hashes
UNION ALL SELECT 'delegations', COUNT(*) FROM delegations;"
```

## Roadmap

- [x] Stage 1: Basic parallel processing (800+ blocks/sec)
- [x] Complete TODO implementations for all data extraction
- [ ] Stage 2: Enhanced parallelism (target 2000+ blocks/sec)
- [ ] Stage 3: Distributed processing across multiple nodes
- [ ] GraphQL API layer
- [ ] Real-time WebSocket subscriptions

## License

Apache License 2.0

## Support

For issues and support:
- GitHub Issues: [your-repo]/nectar/issues
- Documentation: [your-repo]/nectar/wiki