# Nectar

Cardano blockchain indexer built for TiDB.

## What it does

Indexes the entire Cardano blockchain into TiDB for fast queries. Processes all eras from Byron to Conway.

## Why TiDB

- Horizontal scaling by adding TiKV nodes
- No single-node I/O bottleneck
- Distributed storage and query execution
- Linear performance scaling with hardware

## Performance

- 50-500 blocks/second depending on era density
- 45% mainnet synced (7.2M blocks, 40M transactions)
- 89GB indexed data, 481GB TiKV storage

## Requirements

- TiDB cluster (tested with 3 TiKV nodes)
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
dsn = "root:password@tcp(localhost:4000)/nectar"

[cardano]  
node_socket = "/path/to/node.socket"
network_magic = 764824073
```

## Run

```bash
./nectar
```

## Architecture

- Connects to Cardano node via Ouroboros protocol
- Parallel workers process blocks concurrently
- Batch operations optimized for TiDB
- Hash-based keys for distributed writes
- No coordination between multiple instances (yet)