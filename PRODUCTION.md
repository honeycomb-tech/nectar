# Nectar Production Guide

## System Requirements

- **CPU**: 8+ cores (16+ recommended)
- **RAM**: 16GB minimum (32GB+ recommended)
- **Storage**: 2TB+ NVMe SSD for mainnet
- **OS**: Linux (Ubuntu 20.04+ or RHEL 8+)
- **Dependencies**: Go 1.21+, TiDB 7.5+, Cardano node

## Configuration

### Required Environment Variables

```bash
# Database connection (REQUIRED)
TIDB_DSN="user:password@tcp(host:port)/nectar?charset=utf8mb4&parseTime=True&loc=Local"

# Cardano node (auto-detected if not set)
CARDANO_NODE_SOCKET="/path/to/node.socket"
CARDANO_NETWORK_MAGIC="764824073"  # Mainnet
```

### Performance Tuning

```bash
# Optimize for initial sync
BULK_MODE_ENABLED=true          # Batch operations for speed
WORKER_COUNT=16                 # Match CPU cores
DB_CONNECTION_POOL=32           # 2x worker count
BULK_FETCH_RANGE_SIZE=5000      # Larger batches

# Production settings
METRICS_ENABLED=true
METRICS_PORT=9090
LOG_LEVEL=info
```

## Deployment Options

### 1. Systemd Service (Recommended)

```bash
# Install service
sudo mkdir -p /opt/nectar /etc/nectar
sudo cp nectar /opt/nectar/
sudo cp -r migrations /opt/nectar/
sudo cp scripts/nectar.service /etc/systemd/system/
sudo cp scripts/nectar.env.example /etc/nectar/nectar.env

# Configure
sudo nano /etc/nectar/nectar.env

# Start
sudo systemctl daemon-reload
sudo systemctl enable --now nectar
```

### 2. Docker

```bash
# Using docker-compose
docker-compose up -d

# Or standalone
docker run -d \
  --name nectar \
  --env-file .env \
  -v /path/to/node.socket:/node.socket \
  nectar:latest
```

### 3. Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nectar
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: nectar
        image: nectar:latest
        envFrom:
        - secretRef:
            name: nectar-env
        resources:
          requests:
            memory: "16Gi"
            cpu: "8"
          limits:
            memory: "32Gi"
            cpu: "16"
```

## Security Checklist

- [ ] Database credentials in secure store (not git)
- [ ] TLS enabled for database connections
- [ ] Restricted database user permissions
- [ ] Firewall rules configured
- [ ] Resource limits set (systemd/k8s)
- [ ] Log rotation configured
- [ ] Monitoring alerts configured

## Monitoring

### Prometheus Metrics

Exposed at `http://localhost:9090/metrics`:
- `nectar_blocks_processed_total`
- `nectar_sync_height`
- `nectar_processing_speed`
- `nectar_errors_total`

### Grafana Dashboard

Import dashboard from `monitoring/grafana-dashboard.json`

### Health Checks

```bash
# Service status
systemctl status nectar

# Sync progress
curl -s localhost:9090/metrics | grep nectar_sync_height

# Database check
mysql -h $DB_HOST -u $DB_USER -p -e "
  SELECT slot_no, block_no, epoch_no 
  FROM nectar.blocks 
  ORDER BY id DESC LIMIT 1;"
```

## Troubleshooting

### Common Issues

1. **Slow sync speed**
   - Enable `BULK_MODE_ENABLED=true`
   - Increase `WORKER_COUNT` and `DB_CONNECTION_POOL`
   - Check TiDB performance metrics

2. **Database connection errors**
   - Verify `TIDB_DSN` format
   - Check network connectivity
   - Ensure database exists

3. **High memory usage**
   - Normal during initial sync
   - Reduce `WORKER_COUNT` if needed
   - Monitor with `htop` or metrics

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug ./nectar

# Check error logs
tail -f /opt/nectar/logs/error.log
journalctl -u nectar -f
```

## Maintenance

### Backup

```bash
# Backup database (use BR tool for TiDB)
br backup full --pd $PD_ADDR --storage "s3://backup/nectar"

# Backup checkpoints only
mysqldump -h $DB_HOST -u $DB_USER -p nectar checkpoints > checkpoints.sql
```

### Updates

```bash
# Stop service
sudo systemctl stop nectar

# Backup current binary
sudo cp /opt/nectar/nectar /opt/nectar/nectar.bak

# Deploy new binary
sudo cp nectar /opt/nectar/

# Run migrations if needed
cd /opt/nectar && ./nectar -migrate-only

# Start service
sudo systemctl start nectar
```

## Performance Optimization

### TiDB Tuning

```sql
-- Optimize for write-heavy workload
SET GLOBAL tidb_distsql_scan_concurrency = 30;
SET GLOBAL tidb_index_lookup_concurrency = 8;
SET GLOBAL tidb_hash_join_concurrency = 8;

-- Enable async commit
SET GLOBAL tidb_enable_async_commit = ON;
SET GLOBAL tidb_enable_1pc = ON;
```

### Resource Allocation

- **Initial Sync**: Maximum resources, bulk mode enabled
- **Catch-up Mode**: Moderate resources, standard mode
- **Real-time**: Minimal resources, focus on latency

## Support

- Issues: https://github.com/your-repo/nectar/issues
- Documentation: https://github.com/your-repo/nectar/wiki
- Monitoring: Check Grafana dashboards first