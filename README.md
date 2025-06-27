# Nectar

> ⚠️ **UNDER HEAVY DEVELOPMENT** - Currently syncing Mary era. Not production-ready.

Cardano blockchain indexer built for TiDB's distributed architecture.

## Why TiDB for Cardano?

The real power isn't the indexer - it's what TiDB enables:

- **Linear Scaling**: Add more TiKV nodes = proportionally faster queries and writes
- **No I/O Bottlenecks**: Distributed storage eliminates single-disk constraints  
- **Built-in Sharding**: Data automatically distributed by primary keys
- **MySQL Compatible**: Use standard SQL tools and libraries
- **HTAP Capabilities**: TiFlash enables real-time analytics on blockchain data

## Current Sync Status

- **Byron Era**: ✅ Complete (4.5M blocks)
- **Shelley Era**: ✅ Complete (12M blocks)
- **Allegra Era**: ✅ Complete (6.5M blocks)
- **Mary Era**: 🔄 In Progress (currently at block 5.2M)
- **Total Indexed**: 5.2M+ blocks, 8M+ transactions

## What Nectar Does

Nectar is simply a bridge that:
1. Connects to your Cardano node via local socket
2. Reads blocks using the Ouroboros protocol
3. Writes the data to TiDB

The magic happens in TiDB - not the indexer.

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

## Features

- **Web Dashboard**: Real-time monitoring at http://localhost:8080
- **Automatic Resume**: Picks up from last processed block after restart
- **Era-Aware Processing**: Automatically adjusts batch sizes per era
- **Error Filtering**: Ignores non-critical TiDB retry errors

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

## Next Steps

- Complete initial sync to chain tip
- Enable metadata fetching
- Add TiFlash replicas for analytics
- Build query APIs on top of TiDB