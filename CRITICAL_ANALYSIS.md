# Nectar TiDB Performance Crisis Analysis & Recovery Plan

## Current Critical State (as of 22:55)

### 1. System Resources - CRITICAL
- **RAM**: 123GB/124GB used (99.2% - CRITICAL)
- **Swap**: 4GB/4GB used (100% - CRITICAL) 
- **TiKV Memory**: ~93GB combined (3 instances @ ~31GB each)
- **TiDB Memory**: ~11GB combined (3 instances)
- **TiFlash**: ~7.5GB
- **Nectar**: Only 66MB (minimal footprint)

### 2. Performance Metrics - SEVERE
- **INSERT queries**: 1.7-2.6 seconds (should be <10ms)
- **SELECT queries**: 1.7+ seconds (should be <1ms)
- **ROLLBACK**: 6.3 seconds (indicates transaction failures)
- **SELECT 1**: 1 second (basic health check failing)

### 3. TiKV CPU Usage - EXTREME
- TiKV instances: 101-104% CPU each (300%+ combined)
- TiDB instances: 53-58% CPU each
- PD: 31% CPU
- **Total DB CPU**: ~500% (5 full cores constantly)

## Root Cause Analysis

### Primary Issue: TiKV Write Amplification Crisis
1. **Memory Exhaustion**: TiKV is using 93GB RAM for what should be a small dataset
2. **CPU Saturation**: 100%+ CPU per TiKV node indicates severe write amplification
3. **Transaction Failures**: 6-second rollbacks suggest region conflicts or lock contention
4. **Connection Health**: Even "SELECT 1" taking 1 second indicates severe system stress

### Why INSERT IGNORE Isn't Helping
- Our INSERT IGNORE is working at the SQL level
- But TiKV still processes the full write path before rejecting duplicates
- Each duplicate attempt causes:
  - Lock acquisition
  - Transaction coordination
  - Write amplification in RocksDB
  - Memory allocation for pending writes

## Immediate Action Plan

### Phase 1: Emergency Stabilization (Tonight)
1. **Stop Nectar immediately**
   ```bash
   pkill nectar
   ```

2. **Restart TiDB cluster to clear memory**
   ```bash
   tiup cluster stop nectar-tidb
   # Wait 30 seconds
   tiup cluster start nectar-tidb
   ```

3. **Verify clean state**
   - Check memory is freed
   - Ensure swap is cleared
   - Confirm TiKV CPU drops to idle

### Phase 2: Database Optimization (Before Restart)

1. **Add Missing Indexes**
   ```sql
   -- Critical for duplicate detection
   CREATE UNIQUE INDEX idx_blocks_hash ON blocks(hash);
   CREATE INDEX idx_blocks_slot_no ON blocks(slot_no);
   CREATE INDEX idx_blocks_block_no ON blocks(block_no);
   
   -- For slot leader lookups
   CREATE UNIQUE INDEX idx_slot_leaders_hash ON slot_leaders(hash);
   ```

2. **Tune TiKV for Write-Heavy Workload**
   ```toml
   # tikv.toml modifications
   [rocksdb]
   max-background-jobs = 8
   max-background-flushes = 4
   
   [rocksdb.defaultcf]
   block-cache-size = "10GB"
   write-buffer-size = "256MB"
   max-write-buffer-number = 8
   
   [raftstore]
   region-split-check-diff = "32MB"
   region-compact-check-interval = "5m"
   
   [coprocessor]
   region-max-size = "384MB"
   region-split-size = "256MB"
   ```

3. **Reduce TiDB Connection Pool**
   ```toml
   # tidb.toml
   [performance]
   max-procs = 4
   stmt-count-limit = 5000
   txn-total-size-limit = 104857600
   ```

### Phase 3: Application Changes

1. **Implement Bloom Filter for Duplicates**
   - Load existing block hashes into memory bloom filter on startup
   - Check bloom filter before attempting INSERT
   - Dramatically reduce duplicate INSERT attempts

2. **Add Circuit Breaker**
   - Monitor query latency
   - If >3 queries take >1 second, pause for 10 seconds
   - Prevent cascade failures

3. **Batch Deduplication**
   - Collect 100 blocks in memory
   - Single query to check which exist
   - Only INSERT truly new blocks

## Recommended Architecture Change

### Option 1: Switch to PostgreSQL (Recommended)
- Single-node PostgreSQL can handle 1000+ inserts/second
- Native duplicate handling with ON CONFLICT
- 10x less resource usage
- Proven with other Cardano indexers

### Option 2: TiDB Cluster Redesign
- Separate TiKV nodes on different machines
- NVMe storage for write-ahead logs
- Minimum 256GB RAM for cluster
- Dedicated network for Raft consensus

### Option 3: Hybrid Approach
- PostgreSQL for real-time indexing
- Periodic ETL to TiDB for analytics
- Best of both worlds

## Recovery Timeline

**Tonight (2-3 hours)**
1. Stop Nectar
2. Restart TiDB cluster
3. Add critical indexes
4. Implement bloom filter
5. Test with 1 worker, small batches

**Tomorrow**
1. Monitor performance metrics
2. Tune based on observations
3. Gradually increase workers if stable

**This Week**
1. Evaluate PostgreSQL migration
2. Implement proper monitoring
3. Document operational procedures

## Success Metrics
- INSERT latency <50ms
- No swap usage
- TiKV memory <30GB total
- Sustainable 50+ blocks/second

## Critical Decision Point
Given the severity of the issues and fundamental mismatch between TiDB's distributed architecture and our single-node deployment, I strongly recommend:

**Immediate**: Implement emergency fixes to stabilize
**This Week**: Begin PostgreSQL migration
**Long Term**: Use TiDB only for analytics, not real-time indexing

The current setup is fundamentally unsuitable for a blockchain indexer's write patterns.