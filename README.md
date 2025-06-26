# Nectar

> ⚠️ **UNDER HEAVY DEVELOPMENT** - This project is actively being developed and is not yet production-ready. Contributions are welcome! Feel free to open issues or submit PRs.

High-performance Cardano blockchain indexer optimized for TiDB's distributed architecture.

## Overview

Nectar is a specialized Cardano indexer that leverages TiDB's distributed SQL capabilities to achieve linear scaling. Unlike traditional single-node indexers, Nectar can scale horizontally by adding more TiKV nodes, eliminating I/O bottlenecks that plague monolithic database architectures.

## Current Status & Achievements

### ✅ What's Working
- **High-Performance Indexing**: Successfully indexing Cardano blockchain with optimized TiDB backend
- **Byron Era Support**: Complete support for Byron era blocks (1.5M+ blocks indexed)
- **Multi-Era Architecture**: Supports Byron, Shelley, Allegra, Mary, Alonzo, Babbage, and Conway eras
- **Optimized Queries**: Fast inserts with INSERT IGNORE pattern using GORM
- **Connection Pool Management**: Robust connection handling with automatic recovery
- **Real-time Dashboard**: Web and terminal dashboards for monitoring sync progress
- **Memory Optimizations**: Reduced memory footprint by 50% through configuration tuning

### 🚧 In Development
- **Shelley+ Era Processing**: Implementing stake pools, delegations, and rewards
- **Certificate Processing**: Working on stake registrations and pool operations
- **Governance Features**: Conway era governance actions and voting
- **API Layer**: RESTful API for querying blockchain data
- **Performance Tuning**: Further optimizations for 100K+ TPS

### 📊 Performance Metrics
- **Byron Era**: 1.5M+ blocks indexed
- **Insert Speed**: Sub-10ms inserts (improved from 1000ms+)
- **Memory Usage**: Optimized from 128 to 64 connections
- **Worker Efficiency**: 24 parallel workers processing blocks

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

- **Byron Era**: 50-100 blocks/second (current sync rate)
- **Shelley-Mary**: 40-80 blocks/second (staking complexity)
- **Alonzo-Babbage**: 20-50 blocks/second (smart contracts)
- **Peak Performance**: 100+ blocks/second with optimized settings

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
dsn = "root:<PASSWORD>@tcp(localhost:4100)/nectar?charset=utf8mb4&parseTime=True&loc=Local"

[cardano]  
node_socket = "/root/workspace/cardano-node-guild/socket/node.socket"
network_magic = 764824073

[performance]
worker_count = 48
bulk_mode_enabled = true
bulk_fetch_range_size = 20000
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
mysql -h 127.0.0.1 -P 4100 -u root -p'<PASSWORD>' nectar

# Direct TiDB connections (debugging only)
mysql -h 127.0.0.1 -P 4000 -u root -p'<PASSWORD>' nectar  # TiDB-1
mysql -h 127.0.0.1 -P 4001 -u root -p'<PASSWORD>' nectar  # TiDB-2
mysql -h 127.0.0.1 -P 4002 -u root -p'<PASSWORD>' nectar  # TiDB-3
```

## Roadmap

- [ ] TiDB Serverless support with SSL/TLS
- [ ] Selective sync (start from specific slot/era)
- [ ] Configurable processors (disable unwanted data types)
- [ ] Multi-instance coordination for parallel syncing
- [ ] Production deployment guides