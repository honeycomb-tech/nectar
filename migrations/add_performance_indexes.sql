-- Performance indexes identified from bottleneck analysis
-- These will significantly improve stake address lookup performance

-- Index for stake address hash lookups (most critical)
CREATE INDEX IF NOT EXISTS idx_stake_addresses_hash_lookup ON stake_addresses(hash_raw);

-- Optimized index for transaction lookups by block
CREATE INDEX IF NOT EXISTS idx_txes_block_id_optimized ON txes(block_id, id);

-- Composite index for tx_out stake address lookups
CREATE INDEX IF NOT EXISTS idx_tx_outs_tx_stake_lookup ON tx_outs(tx_id, stake_address_id);

-- Additional performance indexes
CREATE INDEX IF NOT EXISTS idx_blocks_slot_no ON blocks(slot_no);
CREATE INDEX IF NOT EXISTS idx_tx_outs_address ON tx_outs(address);
CREATE INDEX IF NOT EXISTS idx_tx_ins_tx_out ON tx_ins(tx_out_id); 