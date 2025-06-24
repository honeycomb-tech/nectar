<<<<<<< HEAD
=======
# Nectar - Distributed Cardano Blockchain Indexer for TiDB

## Overview

Nectar is a purpose-built Cardano blockchain indexer designed specifically for TiDB's distributed architecture. Unlike traditional single-node indexers that hit scalability walls, Nectar leverages TiDB's horizontal scalability to achieve linear performance scaling across multiple nodes while maintaining ACID guarantees.

## Why TiDB?

### The Problem with Traditional Indexers
- **Single-node bottlenecks**: PostgreSQL-based indexers hit I/O and CPU limits
- **Vertical scaling limits**: Can't scale beyond single machine capabilities  
- **Long sync times**: Months to sync mainnet on conventional databases
- **Query performance degradation**: Slows as data grows beyond memory

### TiDB's Distributed Architecture Benefits
- **Horizontal scalability**: Add TiKV nodes to increase throughput
- **Parallel processing**: Distributed transactions across multiple nodes
- **Auto-sharding**: Data automatically distributed across TiKV stores
- **Online scaling**: Add capacity without downtime
- **SQL compatibility**: MySQL protocol with distributed execution

## Architecture

### Nectar Design Principles
1. **Bulk operations first**: Leverages TiDB's distributed write capabilities
2. **Parallel workers**: Multiple goroutines with dedicated connections
3. **Natural key design**: Avoids auto-increment bottlenecks
4. **Batch processing**: Minimizes round trips to database

### Performance Characteristics
- **Byron Era**: ~500 blocks/second (sparse transactions)
- **Shelley Era**: ~200 blocks/second (delegation heavy)
- **Alonzo Era**: ~50 blocks/second (smart contracts)
- **Linear scaling**: Add TiKV nodes to increase throughput

## Technical Implementation

### Database Schema
- **68 tables**: Complete Cardano data model
- **Natural primary keys**: Hash-based keys for distributed performance
- **Composite indexes**: Optimized for common query patterns
- **No foreign keys**: Avoids distributed transaction overhead

### Key Features
- **Era-aware processing**: Optimized paths for each Cardano era
- **Idempotent operations**: Safe for parallel execution
- **Automatic retries**: Handles transient distributed conflicts
- **Memory-efficient**: Streams data without loading full blocks

### TiDB-Specific Optimizations
```sql
-- Distributed query optimization
SET GLOBAL tidb_distsql_scan_concurrency = 50;
SET GLOBAL tidb_executor_concurrency = 16;

-- Async commit for write performance  
SET GLOBAL tidb_enable_async_commit = 1;
SET GLOBAL tidb_enable_1pc = 1;
```

## Deployment Architecture

### Single-Node Development
```toml
[database]
dsn = "root:password@tcp(127.0.0.1:4000)/nectar"
connection_pool = 32

[performance]
worker_count = 28
bulk_fetch_range_size = 10000
```

### Multi-Node Production
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Nectar #1  │     │  Nectar #2  │     │  Nectar #3  │
│  (Node A)   │     │  (Node B)   │     │  (Node C)   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────┴──────┐
                    │  TiDB SQL   │
                    │  (Layer)    │
                    └──────┬──────┘
                           │
       ┌───────────────────┼───────────────────┐
       │                   │                   │
┌──────┴──────┐     ┌──────┴──────┐     ┌──────┴──────┐
│   TiKV #1   │     │   TiKV #2   │     │   TiKV #3   │
│  (Storage)  │     │  (Storage)  │     │  (Storage)  │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Scaling Strategy
1. **Vertical**: Increase workers per Nectar instance
2. **Horizontal**: Add Nectar instances on different nodes
3. **Storage**: Add TiKV nodes for capacity/throughput

## Performance Metrics

### Current Mainnet Sync (45% complete)
- **Blocks indexed**: 7.2M
- **Transactions**: 40.2M  
- **Database size**: 89GB
- **TiKV usage**: 481GB (3 nodes, ~160GB each)

### Resource Utilization
- **CPU**: 60-80% across 32 threads
- **Memory**: 5-6GB per Nectar instance
- **Disk I/O**: 200-300MB/s sustained writes
- **Network**: Minimal (local Cardano node)

## Advantages Over Traditional Indexers

1. **Scalability**: Not limited by single machine I/O
2. **Fault tolerance**: TiKV replication handles node failures
3. **Query performance**: Distributed query execution
4. **Operational simplicity**: No manual sharding required
5. **Future proof**: Can scale with Cardano's growth

## Requirements

- **TiDB Cluster**: Version 7.5.1+
- **Cardano Node**: Local node with socket access
- **Hardware**: 8+ cores, 16GB+ RAM per Nectar instance
- **Storage**: Depends on TiKV cluster configuration

## Building

```bash
go build -o nectar .
```

## Configuration

Create `nectar.toml`:
```toml
[database]
dsn = "root:password@tcp(tidb-host:4000)/nectar"

[cardano]
node_socket = "/path/to/node.socket"
network_magic = 764824073  # mainnet

[performance]
worker_count = 20
bulk_mode_enabled = true
```

## Running

```bash
./nectar
```

## Future Enhancements

- **Distributed coordination**: Redis/etcd for work distribution
- **Checkpoint system**: Resume from specific points
- **Read replicas**: Separate query traffic from indexing
- **Partitioning**: Time-based partitions for historical data

## License

[License details]
>>>>>>> 3c2592f (Major cleanup and optimization)
