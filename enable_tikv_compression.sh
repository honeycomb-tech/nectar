#!/bin/bash
# Enable compression in TiKV for all column families
# This will compress data at the storage level

echo "Enabling TiKV compression for all data..."
echo "This requires updating TiKV configuration and restarting the cluster"
echo ""

# Create new TiKV configuration with compression enabled
cat > /tmp/tikv_compression.toml << 'EOF'
# Add compression settings to existing configuration

[rocksdb.defaultcf]
# Enable compression for default column family (main data)
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
bottommost-level-compression = "zstd"
compression-opts = 3
bottommost-compression-opts = 3

[rocksdb.writecf]
# Enable compression for write column family
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
bottommost-level-compression = "zstd"

[rocksdb.lockcf]
# Enable compression for lock column family
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]

[raftdb.defaultcf]
# Enable compression for raft logs
compression-per-level = ["no", "no", "lz4", "lz4", "lz4", "zstd", "zstd"]
EOF

echo "New compression configuration created."
echo ""
echo "To apply this configuration:"
echo "1. Stop Nectar"
echo "2. Use tiup to update TiKV configuration:"
echo "   tiup cluster edit-config nectar-tidb"
echo "3. Add the compression settings under each rocksdb section"
echo "4. Apply changes:"
echo "   tiup cluster reload nectar-tidb -R tikv"
echo ""
echo "Alternative: Manual compaction to reclaim space immediately:"
echo "For each TiKV instance, run:"
echo "   tikv-ctl --host 127.0.0.1:20160 compact-cluster"
echo "   tikv-ctl --host 127.0.0.1:20161 compact-cluster"
echo "   tikv-ctl --host 127.0.0.1:20162 compact-cluster"