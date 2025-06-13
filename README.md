# Nectar

High-performance Cardano blockchain indexer optimized for distributed database systems.

## Overview

Nectar is a production-grade Cardano blockchain indexer designed to leverage TiDB's distributed SQL capabilities for high-throughput data processing. It provides comprehensive indexing of all on-chain data including transactions, stake pools, governance actions, and multi-asset information.

## Key Features

- Full support for all Cardano eras (Byron through Conway)
- Distributed processing with TiDB optimization
- Real-time progress monitoring dashboard
- Unified error tracking and recovery system
- Smart connection management with automatic protocol detection
- State query integration for reward calculations
- Checkpoint-based resumable synchronization
- Off-chain metadata fetching for pools and governance

## System Requirements

- Go 1.23 or higher
- TiDB 8.5 or higher
- Minimum 16 CPU cores
- Minimum 64GB RAM
- 1TB+ SSD storage recommended

## Installation

```bash
git clone https://github.com/honeycomb-tech/nectar.git
cd nectar
go build -o nectar .
```

## Configuration

### Database Connection
Set the TiDB connection string via environment variable:
```bash
export TIDB_DSN="root:password@tcp(127.0.0.1:4000)/nectar?charset=utf8mb4&parseTime=True"
```

### Optional Environment Variables
```bash
export SKIP_MIGRATIONS=true     # Skip automatic migrations
export NECTAR_LOG_LEVEL=debug   # Set log level (info, debug, error)
```

## Usage

### Basic Operation
```bash
./nectar
```

The indexer will:
1. Connect to TiDB and run migrations if needed
2. Establish connection to local Cardano node
3. Resume from last checkpoint or start from genesis
4. Display real-time synchronization dashboard

### Fresh Sync
For a complete resynchronization:
```bash
mysql -h127.0.0.1 -P4000 -uroot -pYourPassword -e "DROP DATABASE IF EXISTS nectar; CREATE DATABASE nectar;"
./nectar
```

## Architecture

### Core Components

- **Connection Manager**: Smart protocol detection and connection handling
- **Block Processor**: Parallel processing of blocks and transactions
- **Asset Processor**: Multi-asset and NFT metadata handling
- **Governance Processor**: Conway era governance action processing
- **State Query Service**: Integration with node state for rewards
- **Dashboard Renderer**: Real-time progress visualization

### Database Schema

Nectar uses an optimized schema with careful indexing:
- Partitioned tables for era-based data distribution
- Composite indexes for common query patterns
- Cascade deletes for data integrity
- Optimized foreign key relationships

## Performance

Typical performance on recommended hardware:
- Initial sync: 70-150 blocks/second
- Transaction processing: 500+ transactions/second
- Query latency: <100ms for indexed queries
- Full mainnet sync: 2-3 weeks

## Monitoring

The built-in dashboard displays:
- Era synchronization progress
- Current processing speed and peak rates
- Memory and CPU utilization
- Recent activity feed
- Error monitoring and alerts

## Development

### Building from Source
```bash
go mod download
go build -o nectar .
```

### Running Tests
```bash
go test ./...
```

## Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure Cardano node socket is accessible
2. **Migration Errors**: Check TiDB connection and permissions
3. **Slow Sync**: Verify TiDB performance settings and available resources

### Debug Mode
Enable detailed logging:
```bash
export NECTAR_LOG_LEVEL=debug
./nectar
```

## License

Apache License 2.0

## Contributing

Contributions are welcome. Please ensure:
- Code follows Go best practices
- All tests pass
- Documentation is updated
- Commits are signed

## Support

For issues and feature requests, please use the GitHub issue tracker.