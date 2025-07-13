# TiDB/TiKV Compression Guide for Nectar

## Understanding TiDB Compression

Unlike MySQL, TiDB doesn't support table-level compression (`ROW_FORMAT=COMPRESSED`). Instead, compression happens at the TiKV storage layer using RocksDB.

## Current Situation
- Database size: 489GB
- TiKV data: 1,135GB (3x ~378GB)
- No compression currently enabled

## How TiKV Compression Works

TiKV uses RocksDB which supports multiple compression algorithms:
- **LZ4**: Fast compression/decompression, moderate ratio (default)
- **Zstd**: Better compression ratio, slightly slower
- **Snappy**: Very fast, lower compression ratio
- **No compression**: First two levels for hot data

## Enabling Compression

### Option 1: Update TiKV Configuration (Recommended)

1. Edit the cluster configuration:
```bash
tiup cluster edit-config nectar-cluster
```

2. Add compression settings to each TiKV instance:
```toml
tikv_servers:
  - host: 127.0.0.1
    port: 20160
    config:
      rocksdb.defaultcf.compression-per-level: ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
      rocksdb.defaultcf.bottommost-level-compression: "zstd"
      rocksdb.writecf.compression-per-level: ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
      rocksdb.lockcf.compression-per-level: ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
```

3. Reload the configuration:
```bash
tiup cluster reload nectar-cluster -R tikv
```

### Option 2: Manual Compaction (Immediate Space Reclaim)

Run compaction on each TiKV node:
```bash
# Install tikv-ctl if needed
tiup install tikv-ctl

# Compact each instance
tiup ctl:v8.5.1 tikv --host 127.0.0.1:20160 compact-cluster
tiup ctl:v8.5.1 tikv --host 127.0.0.1:20161 compact-cluster
tiup ctl:v8.5.1 tikv --host 127.0.0.1:20162 compact-cluster
```

## Expected Results

With compression enabled:
- **Space savings**: 40-60% reduction
- **TiKV data**: ~1,135GB → ~500-600GB
- **Performance**: 
  - Writes: 5-10% slower
  - Reads: Often faster (less disk I/O)
  - CPU: Slight increase

## Performance Considerations

1. **Compression happens automatically** for new data
2. **Existing data** needs manual compaction or will compress over time
3. **Hot data** (levels 0-1) stays uncompressed for performance
4. **Cold data** (levels 2+) gets compressed

## Monitoring Compression

Check compression status:
```bash
# View TiKV metrics
tiup cluster display nectar-cluster

# Check disk usage
du -sh /tidb-data/tikv-*
```

## Alternative: Reduce Data Size

If compression isn't enough:
1. **Drop indexes** on rarely-queried columns
2. **Archive old data** (e.g., Byron era)
3. **Reduce retention** for some tables
4. **Use TiFlash** for analytical queries (columnar storage)