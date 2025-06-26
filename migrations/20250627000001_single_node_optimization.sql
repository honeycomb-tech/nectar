-- Single Node TiDB Optimization
-- Automatically applied on startup for maximum performance

-- ============================================
-- OPTIMIZE BLOCK INDEXES
-- ============================================

-- Drop suboptimal indexes
DROP INDEX IF EXISTS idx_block_slot_no ON block;

-- Create descending index for MAX queries
CREATE INDEX IF NOT EXISTS idx_block_slot_desc ON block(slot_no DESC) 
    COMMENT 'Optimized for MAX(slot_no) queries';

-- ============================================
-- OPTIMIZE TRANSACTION INDEXES  
-- ============================================

-- Fix transaction indexes for better join performance
DROP INDEX IF EXISTS idx_tx_block_id_block_index ON tx;
CREATE INDEX IF NOT EXISTS idx_tx_block_hash_index ON tx(block_hash, block_index)
    COMMENT 'Optimized for block hash joins';

-- ============================================
-- UTXO OPTIMIZATION
-- ============================================

-- Create partial indexes for UTXO queries
CREATE INDEX IF NOT EXISTS idx_tx_out_unspent ON tx_out(address_raw)
    WHERE spent_by_tx_hash IS NULL
    COMMENT 'UTXO set by address';

-- ============================================
-- ENABLE COMPRESSION
-- ============================================

-- Enable compression on large tables
ALTER TABLE block ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=8;
ALTER TABLE tx ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=8;
ALTER TABLE tx_out ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=8;
ALTER TABLE tx_in ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=8;

-- ============================================
-- TIFLASH PREPARATION
-- ============================================

-- Enable TiFlash replicas (will activate when TiFlash is available)
ALTER TABLE block SET TIFLASH REPLICA 1;
ALTER TABLE tx SET TIFLASH REPLICA 1;
ALTER TABLE tx_out SET TIFLASH REPLICA 1;
ALTER TABLE pool_stats SET TIFLASH REPLICA 1;
ALTER TABLE epoch_stats SET TIFLASH REPLICA 1;

-- ============================================
-- UPDATE STATISTICS
-- ============================================

-- Update statistics for query optimizer
ANALYZE TABLE block;
ANALYZE TABLE tx;
ANALYZE TABLE tx_out;