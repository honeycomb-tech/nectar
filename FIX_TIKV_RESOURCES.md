# TiKV Resource Configuration Fix

## The Problem
We configured Nectar to use 70% of resources, but TiKV is running with UNLIMITED memory and default settings. It's using 93GB RAM when we planned for ~30GB.

## Quick Fix Commands

### 1. Stop Everything
```bash
pkill nectar
tiup cluster stop nectar-tidb
```

### 2. Create TiKV Resource Limits
```bash
# Create systemd override for each TiKV instance
sudo mkdir -p /etc/systemd/system/tikv-20160.service.d
sudo mkdir -p /etc/systemd/system/tikv-20161.service.d
sudo mkdir -p /etc/systemd/system/tikv-20162.service.d

# Limit each TiKV to 10GB (30GB total)
cat << 'EOF' | sudo tee /etc/systemd/system/tikv-20160.service.d/memory.conf
[Service]
MemoryMax=10G
MemoryHigh=9G
CPUQuota=100%
EOF

# Copy to other instances
sudo cp /etc/systemd/system/tikv-20160.service.d/memory.conf /etc/systemd/system/tikv-20161.service.d/
sudo cp /etc/systemd/system/tikv-20160.service.d/memory.conf /etc/systemd/system/tikv-20162.service.d/
```

### 3. Configure TiKV Memory Settings
```bash
# Create custom tikv.toml with memory limits
cat << 'EOF' > /tmp/tikv-memory.toml
[rocksdb.defaultcf]
block-cache-size = "2GB"  # Was probably 20GB+ per instance

[rocksdb.writecf]
block-cache-size = "512MB"

[rocksdb.lockcf]
block-cache-size = "256MB"

[storage]
block-cache-size = "2GB"

[raftdb]
max-total-wal-size = "1GB"

[server]
grpc-memory-pool-quota = "2GB"
EOF

# Apply to all TiKV instances
tiup cluster edit-config nectar-tidb
# Add the above settings under tikv_servers section
```

### 4. Restart with Limits
```bash
sudo systemctl daemon-reload
tiup cluster reload nectar-tidb
```

## Alternative: Docker/Cgroup Limits
If TiKV is running in containers:
```bash
# Find TiKV processes
docker ps | grep tikv

# Update each container with memory limits
docker update --memory="10g" --memory-swap="10g" <container_id>
```

## The Real Issue
TiDB/TiKV assumes it owns the entire machine. On a shared system, we MUST set explicit limits or it will consume everything available.

Our 70/30 plan was correct, but we only applied it to Nectar, not the database itself!