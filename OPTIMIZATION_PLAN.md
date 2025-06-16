# Nectar Indexer Optimization Plan

## Overview
This plan outlines practical optimizations for the existing Nectar indexer code without requiring a full rewrite. Each optimization can be implemented incrementally and tested independently.

## Current Performance Baseline
- Processing speed: ~226 blocks/sec (Byron era)
- Sequential processing only
- Single database connection used
- Small batch sizes

## Quick Wins (Implement First)

### 1. Increase Batch Sizes
**File:** `processors/block_processor.go`
**Line:** 157
```go
// Current
batchSize := 50

// Optimized
batchSize := 500  // 10x improvement for TiDB
```

### 2. Enable More TiDB Optimizations
**File:** `database/tidb.go`
**Function:** `enableTiDBOptimizations`
```go
// Add these session settings
"SET SESSION tidb_enable_vectorized_expression = ON",
"SET SESSION tidb_projection_concurrency = 8", 
"SET SESSION tidb_hash_join_concurrency = 8",
"SET SESSION tidb_enable_parallel_apply = ON",
"SET SESSION tidb_enable_batch_dml = ON",
```

### 3. Use Connection Pool Round-Robin
**File:** `main.go`
**Lines:** 435-447
```go
// Add at struct level
processorIndex uint64

// In processBlock method
processorIndex := atomic.AddUint64(&rai.processorIndex, 1) % uint64(len(rai.dbConnections))
processor := rai.dbConnections[processorIndex]
```

## Medium Impact Optimizations

### 4. Parallel Transaction Component Processing
**File:** `processors/block_processor.go`
**Function:** `processTransaction`

Create parallel processing for independent components:
```go
func (bp *BlockProcessor) processTransaction(ctx context.Context, tx *gorm.DB, blockHash []byte, blockHeight uint64, blockTime int64, txIdx uint64, transaction TxBody, blockType uint, caCerts *AssetCertsMap) error {
    // Process hash first
    txHash := transaction.Hash()
    
    // Create channels for parallel work
    errChan := make(chan error, 4)
    var wg sync.WaitGroup
    
    // Process components in parallel
    wg.Add(3)
    
    // Inputs
    go func() {
        defer wg.Done()
        if err := bp.processTransactionInputs(ctx, tx, txHash, transaction, blockType); err != nil {
            errChan <- err
        }
    }()
    
    // Outputs
    go func() {
        defer wg.Done()
        if err := bp.processTransactionOutputs(ctx, tx, txHash, transaction, blockType); err != nil {
            errChan <- err
        }
    }()
    
    // Metadata
    go func() {
        defer wg.Done()
        if err := bp.processTransactionMetadata(ctx, tx, txHash, transaction); err != nil {
            errChan <- err
        }
    }()
    
    // Wait and check errors
    go func() {
        wg.Wait()
        close(errChan)
    }()
    
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    // Process certificates (depends on outputs)
    return bp.processTransactionCertificates(ctx, tx, txHash, transaction, blockType, caCerts)
}
```

### 5. Implement Bulk Sync Mode Detection
**File:** `main.go`
**Add to RealtimeCardanoIndexer struct:**
```go
type RealtimeCardanoIndexer struct {
    // ... existing fields ...
    isBulkSync     bool
    bulkSyncSlot   uint64  // Switch to normal mode after this slot
}

// In processBlock method
func (rai *RealtimeCardanoIndexer) processBlock(block ledger.Block, blockType uint) error {
    currentSlot := block.Header().SlotNumber()
    
    // Auto-detect bulk sync mode
    if !rai.isBulkSync && rai.currentSlot > 0 {
        slotGap := currentSlot - rai.currentSlot
        if slotGap > 10000 {
            rai.isBulkSync = true
            rai.bulkSyncSlot = rai.tipSlot - 1000 // Switch to normal 1000 slots from tip
            fmt.Println("Entering bulk sync mode")
        }
    }
    
    // Check if we should exit bulk sync
    if rai.isBulkSync && currentSlot >= rai.bulkSyncSlot {
        rai.isBulkSync = false
        fmt.Println("Exiting bulk sync mode")
    }
    
    // Use appropriate processing strategy
    if rai.isBulkSync {
        return rai.processBulkBlock(block, blockType)
    }
    return rai.processNormalBlock(block, blockType)
}
```

### 6. Bulk Insert Optimization
**File:** `processors/block_processor.go`
**Add bulk insert methods:**
```go
func (bp *BlockProcessor) BulkInsertTxOuts(ctx context.Context, tx *gorm.DB, outputs []models.TxOut) error {
    if len(outputs) == 0 {
        return nil
    }
    
    // For bulk sync, use raw SQL for speed
    valueStrings := make([]string, 0, len(outputs))
    valueArgs := make([]interface{}, 0, len(outputs)*10)
    
    for _, output := range outputs {
        valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
        valueArgs = append(valueArgs, 
            output.TxHash,
            output.Index,
            output.Address,
            output.PaymentCred,
            output.Value,
            output.AddressHasScript,
            output.DataHash,
            output.InlineDatumBytes,
            output.ReferenceScriptBytes,
            output.ConsumedByTxHash,
        )
    }
    
    query := fmt.Sprintf("INSERT IGNORE INTO tx_outs (tx_hash, index, address, payment_cred, value, address_has_script, data_hash, inline_datum_bytes, reference_script_bytes, consumed_by_tx_hash) VALUES %s",
        strings.Join(valueStrings, ","))
    
    return tx.Exec(query, valueArgs...).Error
}
```

## Advanced Optimizations

### 7. Block Prefetching
**File:** `main.go`
**Add prefetcher:**
```go
type BlockPrefetcher struct {
    buffer     chan BlockData
    bufferSize int
}

type BlockData struct {
    block     ledger.Block
    blockType uint
}

func NewBlockPrefetcher(size int) *BlockPrefetcher {
    return &BlockPrefetcher{
        buffer:     make(chan BlockData, size),
        bufferSize: size,
    }
}
```

### 8. Async Error Logging
**File:** `database/error_handler.go`
**Make error logging non-blocking:**
```go
type AsyncErrorLogger struct {
    errorChan chan *models.ProcessingError
    db        *gorm.DB
}

func NewAsyncErrorLogger(db *gorm.DB) *AsyncErrorLogger {
    logger := &AsyncErrorLogger{
        errorChan: make(chan *models.ProcessingError, 1000),
        db:        db,
    }
    go logger.worker()
    return logger
}

func (ael *AsyncErrorLogger) worker() {
    batch := make([]*models.ProcessingError, 0, 100)
    ticker := time.NewTicker(5 * time.Second)
    
    for {
        select {
        case err := <-ael.errorChan:
            batch = append(batch, err)
            if len(batch) >= 100 {
                ael.flush(batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                ael.flush(batch)
                batch = batch[:0]
            }
        }
    }
}
```

## Implementation Schedule

### Week 1: Quick Wins
- [ ] Increase batch sizes
- [ ] Add TiDB optimizations
- [ ] Implement connection pool round-robin
- [ ] Test and measure performance improvement

### Week 2: Parallel Processing
- [ ] Implement parallel transaction component processing
- [ ] Add bulk sync mode detection
- [ ] Test with full sync from genesis

### Week 3: Bulk Operations
- [ ] Implement bulk insert methods
- [ ] Add prefetching for chain sync
- [ ] Optimize error logging

### Week 4: Fine-tuning
- [ ] Profile and identify remaining bottlenecks
- [ ] Tune batch sizes based on metrics
- [ ] Add adaptive performance tuning

## Expected Performance Gains

Based on these optimizations:
- **Quick Wins:** 2-3x improvement (500-700 blocks/sec)
- **Parallel Processing:** Additional 2-3x (1,500-2,000 blocks/sec)
- **Bulk Operations:** Additional 2x for initial sync (3,000-4,000 blocks/sec)

Total expected improvement: **10-20x** current performance

## Monitoring and Rollback

Each optimization should be:
1. Implemented behind a feature flag
2. Tested in isolation
3. Monitored for performance impact
4. Easily reversible if issues arise

## Code Examples

### Feature Flags
```go
type FeatureFlags struct {
    ParallelProcessing  bool
    BulkInserts        bool
    Prefetching        bool
    AsyncLogging       bool
}

var features = FeatureFlags{
    ParallelProcessing: false, // Enable one at a time
    BulkInserts:       false,
    Prefetching:       false,
    AsyncLogging:      false,
}
```

This plan provides a clear path to optimize your indexer incrementally while maintaining stability and data integrity.