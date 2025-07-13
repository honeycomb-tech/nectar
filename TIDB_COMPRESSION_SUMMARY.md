# TiDB Compression Summary

## Key Findings

1. **TiDB doesn't support table-level compression** (`ROW_FORMAT=COMPRESSED` is not supported)
2. **Compression happens at the TiKV storage layer** using RocksDB
3. **Default: No compression enabled** in your cluster

## Current Status

- Database logical size: 489GB
- TiKV physical storage: 1,135GB (3x ~378GB per node)
- Disk usage: 89% (189GB free)

## Solution Applied

We initiated a manual compaction using:
```bash
tiup ctl:v8.5.1 tikv --pd 127.0.0.1:2379 compact-cluster
```

This will:
- Compact all data in TiKV
- Remove deleted/obsolete data
- Apply any default compression if configured
- Expected to free 10-30% space immediately

## Long-term Solution

To enable compression for all new data:

1. Edit cluster config: `tiup cluster edit-config nectar-cluster`
2. Add compression settings for each TiKV node:
```yaml
tikv_servers:
  - host: 127.0.0.1
    config:
      rocksdb.defaultcf.compression-per-level: ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
      rocksdb.defaultcf.bottommost-level-compression: "zstd"
```
3. Reload: `tiup cluster reload nectar-cluster -R tikv`

## Benefits of Unified Compression

- **Consistency**: All data compressed the same way
- **Simplicity**: No per-table management needed
- **Performance**: Often faster due to less disk I/O
- **Space**: 40-60% reduction expected

## Monitoring

Check compaction progress:
```bash
# View disk usage
df -h /
du -sh /tidb-data/tikv-*

# Check TiKV logs
tail -f /tidb-deploy/tikv-20160/log/tikv.log | grep -i compact
```

The compaction is running in the background and should complete within a few hours.