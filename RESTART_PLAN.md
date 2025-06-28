# Nectar Restart and Optimization Plan

## Issues Fixed

1. **TiDB Daily Restart Issue** ✅
   - Removed `/etc/logrotate.d/tidb` which was sending SIGHUP to TiDB at midnight
   - TiDB now handles its own log rotation internally
   - No more daily database restarts

2. **Connection Recovery** ✅
   - Added `ConnectionRecovery` mechanism for automatic reconnection
   - Health checks every 5 seconds
   - Exponential backoff for reconnection attempts
   - Maximum 10 reconnection attempts with up to 30-second delays

## Optimization Plan

### Current State
- **Memory**: Was at 57GB before crash (now cleared)
- **Workers**: 8 workers
- **Connection Pool**: 8 connections per worker
- **Processing**: Stopped at slot 60,549,573 (Alonzo era)

### Recommended New Configuration

Based on fixing the memory leak and having stable operations:

```bash
# Increased processing power
export WORKER_COUNT=12              # Up from 8
export DB_CONNECTION_POOL=10        # Up from 8
export BULK_FETCH_RANGE_SIZE=3000   # Up from 2000

# Memory management
export GOGC=100                     # Default Go GC
export GOMEMLIMIT=100GiB           # Set hard memory limit

# Connection resilience
export DB_CONN_MAX_LIFETIME=10m    # Shorter connection lifetime
export DB_CONN_MAX_IDLE_TIME=5m    # Shorter idle time
```

### Performance Expectations

With the optimizations:
- **Alonzo Era**: 100-200 blocks/sec (up from 50-100)
- **Memory Usage**: Should stabilize around 60-80GB
- **Connection Stability**: Automatic recovery from database restarts

## Restart Procedure

### 1. Pre-Restart Checks

```bash
# Verify TiDB is healthy
mysql -h 127.0.0.1 -P 4100 -u root -e "SELECT 'TiDB is running' as status;"

# Check TiDB won't restart at midnight anymore
ls -la /etc/logrotate.d/tidb*
# Should show: tidb.disabled (not tidb)

# Clear old logs to free space
cd /root/workspace/cardano-stack/Nectar
rm -f errors.log.*.bak unified_errors.log.*.bak
```

### 2. Start Nectar with New Configuration

```bash
cd /root/workspace/cardano-stack/Nectar

# Set optimized configuration
export WORKER_COUNT=12
export DB_CONNECTION_POOL=10
export BULK_FETCH_RANGE_SIZE=3000
export GOGC=100
export GOMEMLIMIT=100GiB

# Start with connection recovery enabled
./nectar
```

### 3. Monitor Initial Performance

Watch for:
- Connection recovery messages: `[ConnectionRecovery] Database connection established successfully`
- Worker startup: Should see 12 workers initializing
- Memory usage: Should grow gradually, not spike
- Processing speed: Should see improved blocks/sec in Alonzo

### 4. Long-Term Monitoring

Create a monitoring script:

```bash
#!/bin/bash
# monitor_nectar.sh

while true; do
    echo "=== $(date) ==="
    
    # Check Nectar process
    ps aux | grep nectar | grep -v grep | awk '{print "CPU:", $3"%, MEM:", $4"%, RSS:", $6/1024/1024"GB"}'
    
    # Check TiDB connections
    mysql -h 127.0.0.1 -P 4100 -u root -e "SHOW PROCESSLIST;" 2>/dev/null | wc -l | awk '{print "Active DB connections:", $1-1}'
    
    # Check latest block
    mysql -h 127.0.0.1 -P 4100 -u root nectar -e "SELECT MAX(slot_no) as latest_slot, MAX(block_no) as latest_block FROM block;" 2>/dev/null
    
    echo ""
    sleep 60
done
```

## Expected Outcomes

1. **No More Midnight Crashes**: TiDB will run continuously
2. **Automatic Recovery**: If TiDB does restart, Nectar will reconnect automatically
3. **Better Performance**: 50% more workers and optimized batching
4. **Stable Memory**: With GOMEMLIMIT set, no more unbounded growth
5. **Faster Alonzo Sync**: Should complete Alonzo in ~24-36 hours instead of 48-72

## Emergency Procedures

If issues occur:

1. **High Memory Usage**:
   ```bash
   # Reduce workers temporarily
   export WORKER_COUNT=8
   # Restart Nectar
   ```

2. **Database Connection Issues**:
   - Check TiDB logs: `tail -f /tidb-deploy/tidb-4000/log/tidb.log`
   - Verify TiDB is running: `systemctl status tidb-4000`
   - Manual TiDB restart if needed: `systemctl restart tidb-4000`

3. **Slow Processing**:
   - Check for lock contention: `mysql -h 127.0.0.1 -P 4100 -u root -e "SHOW PROCESSLIST;"`
   - Reduce batch size: `export BULK_FETCH_RANGE_SIZE=1000`

## Success Metrics

- ✅ No database disconnections for 24+ hours
- ✅ Memory usage stable below 100GB
- ✅ Processing speed >100 blocks/sec in Alonzo
- ✅ Automatic recovery from any connection issues
- ✅ Complete Alonzo era without manual intervention