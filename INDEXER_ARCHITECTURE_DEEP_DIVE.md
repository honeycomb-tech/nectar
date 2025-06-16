# Nectar Indexer: Deep Architecture Analysis & Optimization Plan

## Current Architecture State

### 1. Write-Heavy vs Read-Heavy Design

**Current State: Write-Optimized Only**
- The indexer is currently optimized ONLY for write-heavy operations
- No automatic switch to read-heavy mode when reaching tip
- Same processing path for both initial sync and tip following

**What Should Happen:**
```
Initial Sync (Write-Heavy Mode):
- Bulk inserts with large batches
- Minimal indexes active
- Foreign keys disabled
- Aggressive caching
- Maximum parallelism

Tip Following (Read-Heavy Mode):
- Smaller batches (1-10 blocks)
- All indexes active
- Foreign keys enforced
- Query optimization priority
- Real-time consistency
```

### 2. Parallelism Analysis

**Current State: SEQUENTIAL Processing**
```go
// main.go - Single-threaded chain sync
rollforwardFunc := func(blockType uint, blockData interface{}, tip chainsync.Tip) error {
    // Process one block at a time
    return indexer.processBlock(blockType, blockData, tip)
}
```

**Missing Parallelism Opportunities:**
1. **Block-level parallelism**: Could process multiple blocks concurrently
2. **Transaction-level parallelism**: Within a block, transactions are independent
3. **Component parallelism**: Different processors could run in parallel
4. **Pipeline parallelism**: Fetch next while processing current

### 3. Foreign Keys Status

**Current Implementation:**
```sql
-- Foreign keys are DISABLED during migration
SET foreign_key_checks = 0
-- Tables created without FK constraints
DisableForeignKeyConstraintWhenMigrating: true
```

**Why Foreign Keys Matter:**
1. **Data Integrity**: Prevent orphaned records
2. **Query Optimization**: Help query planner
3. **Cascading Operations**: Automatic cleanup
4. **Application Logic**: Enforce business rules

**Current Issues:**
- FKs never re-enabled after initial table creation
- No post-sync FK creation script
- Missing many important relationships

## Detailed Optimization Plan

### Phase 1: Dual-Mode Architecture

#### 1.1 Create Mode Detection
```go
type SyncMode int
const (
    ModeBulkSync SyncMode = iota  // Initial sync
    ModeTipFollow                 // Near tip
    ModeTransition               // Switching modes
)

type IndexerMode struct {
    current         SyncMode
    blocksBehind    int
    transitionPoint int  // e.g., 1000 blocks
}
```

#### 1.2 Mode-Specific Configurations
```go
type ModeConfig struct {
    // Bulk Sync Settings
    BulkBatchSizes      BatchConfig
    BulkParallelism     int
    BulkIndexStrategy   IndexStrategy
    
    // Tip Follow Settings  
    TipBatchSizes       BatchConfig
    TipParallelism      int
    TipIndexStrategy    IndexStrategy
}
```

### Phase 2: Implement Parallelism

#### 2.1 Block Pipeline Architecture
```
[Chain Sync] → [Block Queue] → [Block Workers] → [TX Processors] → [DB Writers]
     ↓              ↓                ↓                  ↓              ↓
   Fetch         Buffer          Process           Transform       Commit
```

#### 2.2 Parallel Processing Levels
```go
// Level 1: Multiple blocks in parallel
blockWorkers := 4  // Process 4 blocks concurrently

// Level 2: Multiple transactions per block
txWorkers := 8     // Process 8 TXs concurrently per block

// Level 3: Component parallelism
processors := []Processor{
    assetProcessor,      // Can run in parallel
    certificateProcessor,
    withdrawalProcessor,
    metadataProcessor,
}
```

#### 2.3 Synchronization Requirements
- Blocks must maintain order for chain continuity
- Transactions within a block can be processed in any order
- Outputs must be written before inputs reference them
- Foreign keys require careful ordering

### Phase 3: Foreign Key Strategy

#### 3.1 Initial Sync (No FKs)
```sql
-- Tables created without foreign keys for speed
CREATE TABLE tx_outs (
    tx_hash VARBINARY(32),
    index INT,
    -- No FK to txes table during sync
    PRIMARY KEY (tx_hash, index)
);
```

#### 3.2 Post-Sync FK Addition
```sql
-- After reaching tip, add all foreign keys
ALTER TABLE tx_outs 
ADD CONSTRAINT fk_tx_outs_tx 
FOREIGN KEY (tx_hash) REFERENCES txes(hash);

-- Total of ~50+ foreign keys to add
```

#### 3.3 FK Necessity Analysis
**Critical FKs (Data Integrity):**
- tx_outs → txes (outputs must have valid TX)
- tx_ins → tx_outs (inputs must reference valid outputs)
- blocks → previous_block (chain continuity)

**Performance FKs (Query Optimization):**
- delegations → stake_addresses, pool_hashes
- rewards → stake_addresses
- All governance relationships

**Optional FKs (Nice to Have):**
- Metadata references
- Off-chain data relationships

### Phase 4: Performance Targets

#### 4.1 Initial Sync Performance
```
Current State:
- Byron: ~250-350 blocks/second
- Single-threaded processing
- Sequential I/O

Target State:
- Byron: 2,000+ blocks/second
- 4-8 parallel block processors
- Pipelined I/O
- Bulk loading where possible
```

#### 4.2 Tip Following Performance
```
Current State:
- Same as bulk sync
- Over-optimized for writes
- Poor query performance

Target State:
- <100ms block processing
- All indexes active
- Read-optimized tables
- Real-time query capability
```

### Phase 5: Implementation Roadmap

#### Step 1: Mode Detection (Week 1)
- [ ] Implement sync mode detection
- [ ] Create mode-specific configurations
- [ ] Add mode switching logic
- [ ] Dashboard mode indicator

#### Step 2: Basic Parallelism (Week 2)
- [ ] Implement block queue system
- [ ] Add worker pool for blocks
- [ ] Ensure ordering guarantees
- [ ] Benchmark improvements

#### Step 3: Advanced Parallelism (Week 3)
- [ ] Transaction-level parallelism
- [ ] Component parallelism
- [ ] Pipeline architecture
- [ ] Concurrency testing

#### Step 4: Foreign Key Management (Week 4)
- [ ] Create FK addition script
- [ ] Implement FK state tracking
- [ ] Add FK validation tooling
- [ ] Performance impact analysis

#### Step 5: Mode Optimization (Week 5)
- [ ] Bulk sync optimizations
- [ ] Tip follow optimizations
- [ ] Mode transition handling
- [ ] Performance monitoring

### Current Bottlenecks

1. **Sequential Processing**: Single-threaded is leaving 90% of CPU idle
2. **Small Batches**: Current batches too small for TiDB's capabilities
3. **No Prefetching**: Not fetching ahead while processing
4. **Synchronous I/O**: Blocking on each database operation
5. **No Index Management**: Indexes active during bulk insert

### TiDB-Specific Optimizations Needed

1. **Bulk Loading**:
   ```sql
   LOAD DATA LOCAL INFILE 'blocks.csv' INTO TABLE blocks
   ```

2. **Region Pre-splitting**:
   ```sql
   ALTER TABLE blocks SPLIT REGION 1000;
   ```

3. **Placement Rules**:
   ```sql
   ALTER TABLE hot_table SET TIFLASH REPLICA 1;
   ```

4. **Statistics Management**:
   ```sql
   SET SESSION tidb_build_stats_concurrency = 8;
   ANALYZE TABLE blocks;
   ```

## Summary

**Where We Are:**
- Write-optimized only, no mode switching
- Sequential processing leaving resources idle  
- Foreign keys disabled permanently
- Same code path for bulk and tip

**Where We Need to Be:**
- Dual-mode architecture with automatic switching
- Full parallelism at multiple levels
- Intelligent FK management
- Optimized for both bulk sync and real-time

**Biggest Wins Available:**
1. Parallel block processing: 4-8x speedup
2. Proper batching for TiDB: 2-3x speedup  
3. Pipeline architecture: 1.5-2x speedup
4. Mode-specific optimization: 10x for queries at tip

**Total Potential Improvement: 20-50x current speed**