# Nectar

High-performance Cardano blockchain indexer optimized for TiDB's distributed architecture.

## Overview

Nectar is a specialized Cardano indexer that leverages TiDB's distributed SQL capabilities to achieve linear scaling. Unlike traditional single-node indexers, Nectar can scale horizontally by adding more TiKV nodes, eliminating I/O bottlenecks that plague monolithic database architectures.

## Current Status (June 2025)

- **Sync Progress**: 46.5% complete (7.8M blocks, 51.6M transactions)
- **Current Era**: Babbage (slot 73.8M)
- **Performance**: 50-100 blocks/second average
- **Database Size**: 100GB+ indexed data across 3 TiKV nodes
- **Architecture**: Node-to-Client (N2C) protocol exclusively

## Why TiDB?

Traditional Cardano indexers hit a wall with single-node databases:
- PostgreSQL: I/O bound at ~30-40 blocks/second
- MySQL: Similar limitations with large datasets
- SQLite: Not suitable for production scale

TiDB solves these problems:
- **Horizontal Scaling**: Add TiKV nodes for linear performance gains
- **Distributed Storage**: Data automatically sharded across nodes
- **Parallel Query Execution**: Queries scale with cluster size
- **No Single Point of Failure**: Built-in high availability

## Performance Metrics

- **Byron Era**: 200-500 blocks/second (simple transactions)
- **Shelley-Mary**: 100-200 blocks/second (staking added)
- **Alonzo-Babbage**: 50-100 blocks/second (smart contracts)
- **Peak Performance**: 1000+ blocks/second with sufficient hardware

## Requirements

- TiDB cluster v8.5.1+ (3 TiDB servers, 3 TiKV nodes, 1 TiFlash node)
- HAProxy load balancer for TiDB servers
- Cardano node with local socket
- 8+ CPU cores, 16GB+ RAM

## Build

```bash
go build -o nectar .
```

## Configure

Create `nectar.toml`:
```toml
[database]
# Connect via HAProxy load balancer (port 4100) for high availability
dsn = "root:50bcT*DaeU2-19@+Q4@tcp(localhost:4100)/nectar?charset=utf8mb4&parseTime=True&loc=Local"

[cardano]  
node_socket = "/root/workspace/cardano-node-guild/socket/node.socket"
network_magic = 764824073

[performance]
worker_count = 28
bulk_mode_enabled = true
bulk_fetch_range_size = 10000
```

## Run

```bash
./nectar
```

## Architecture

### Design Principles
- **Node-to-Client Protocol**: Direct connection to Cardano node's UNIX socket
- **Parallel Processing**: Multiple workers process blocks concurrently
- **Batch Operations**: Optimized for TiDB's distributed architecture
- **Hash-Based Sharding**: Transaction and block hashes ensure even distribution
- **Modular Processors**: Separate handlers for each data type (blocks, transactions, certificates, etc.)

### Key Features
- **Web Dashboard**: Real-time monitoring at http://localhost:8080
- **Error Monitoring**: Comprehensive error tracking and filtering
- **Performance Metrics**: Detailed statistics on sync progress
- **Automatic Resume**: Picks up from last processed block after restart

## Known Issues

1. **Script Extraction**: Scripts are not being extracted during initial sync due to a bug that was fixed mid-sync. Requires resync from Alonzo era (slot 72316896) to capture all scripts.

2. **Metadata Fetcher**: Disabled by default during initial sync for performance. Enable after reaching chain tip.

3. **TiDB Serverless**: Not yet supported - requires SSL/TLS configuration additions.

## TiDB Cluster Management

### Cluster Status
```bash
# Check cluster status
tiup cluster display nectar-cluster

# Start/stop cluster
tiup cluster start nectar-cluster
tiup cluster stop nectar-cluster

# Access TiDB Dashboard
http://127.0.0.1:2379/dashboard

# HAProxy stats
http://127.0.0.1:8404/stats
```

### Database Access
```bash
# Connect via HAProxy (recommended)
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' nectar

# Direct TiDB connections (debugging only)
mysql -h 127.0.0.1 -P 4000 -u root -p'50bcT*DaeU2-19@+Q4' nectar  # TiDB-1
mysql -h 127.0.0.1 -P 4001 -u root -p'50bcT*DaeU2-19@+Q4' nectar  # TiDB-2
mysql -h 127.0.0.1 -P 4002 -u root -p'50bcT*DaeU2-19@+Q4' nectar  # TiDB-3
```

## Roadmap

- [ ] TiDB Serverless support with SSL/TLS
- [ ] Selective sync (start from specific slot/era)
- [ ] Configurable processors (disable unwanted data types)
- [ ] Multi-instance coordination for parallel syncing
- [ ] Production deployment guides