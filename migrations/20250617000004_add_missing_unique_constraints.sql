-- Add missing unique constraints for idempotent operations
-- These constraints ensure duplicate key errors are properly handled

-- ============================================
-- TX_IN CONSTRAINTS
-- ============================================

-- Add unique constraint on tx_ins if not exists
ALTER TABLE tx_ins
ADD CONSTRAINT IF NOT EXISTS unique_tx_ins_tx_in_hash_index 
UNIQUE (tx_in_hash, tx_in_index);

-- ============================================
-- TX_OUT CONSTRAINTS
-- ============================================

-- Add unique constraint on tx_outs if not exists
ALTER TABLE tx_outs
ADD CONSTRAINT IF NOT EXISTS unique_tx_outs_tx_hash_index
UNIQUE (tx_hash, `index`);

-- ============================================
-- CERTIFICATE CONSTRAINTS
-- ============================================

-- Add unique constraint on stake_registrations
ALTER TABLE stake_registrations
ADD CONSTRAINT IF NOT EXISTS unique_stake_registrations_tx_cert
UNIQUE (tx_hash, cert_index);

-- Add unique constraint on stake_deregistrations  
ALTER TABLE stake_deregistrations
ADD CONSTRAINT IF NOT EXISTS unique_stake_deregistrations_tx_cert
UNIQUE (tx_hash, cert_index);

-- Add unique constraint on delegations
ALTER TABLE delegations
ADD CONSTRAINT IF NOT EXISTS unique_delegations_tx_cert
UNIQUE (tx_hash, cert_index);

-- Add unique constraint on pool_updates
ALTER TABLE pool_updates
ADD CONSTRAINT IF NOT EXISTS unique_pool_updates_tx_cert
UNIQUE (tx_hash, cert_index);

-- Add unique constraint on pool_retires
ALTER TABLE pool_retires
ADD CONSTRAINT IF NOT EXISTS unique_pool_retires_tx_cert
UNIQUE (tx_hash, cert_index);

-- Add unique constraint on pool_relays
ALTER TABLE pool_relays
ADD CONSTRAINT IF NOT EXISTS unique_pool_relays_update_relay
UNIQUE (update_tx_hash, update_cert_index, relay_index);

-- Add unique constraint on pool_owners
ALTER TABLE pool_owners
ADD CONSTRAINT IF NOT EXISTS unique_pool_owners_update_owner
UNIQUE (update_tx_hash, update_cert_index, owner_hash);

-- ============================================
-- COLLATERAL CONSTRAINTS
-- ============================================

-- Add unique constraint on collateral_tx_in
ALTER TABLE collateral_tx_in
ADD CONSTRAINT IF NOT EXISTS unique_collateral_tx_in
UNIQUE (tx_in_hash, tx_out_hash, tx_out_index);

-- Add unique constraint on collateral_tx_out
ALTER TABLE collateral_tx_out
ADD CONSTRAINT IF NOT EXISTS unique_collateral_tx_out
UNIQUE (tx_hash, `index`);

-- ============================================
-- REFERENCE INPUT CONSTRAINTS
-- ============================================

-- Add unique constraint on reference_tx_in
ALTER TABLE reference_tx_in
ADD CONSTRAINT IF NOT EXISTS unique_reference_tx_in
UNIQUE (tx_in_hash, tx_out_hash, tx_out_index);

-- ============================================
-- METADATA CONSTRAINTS
-- ============================================

-- Add unique constraint on pool_metadata_refs (hash is already primary key)
-- No additional constraint needed

-- ============================================
-- BLOCK CONSTRAINTS
-- ============================================

-- Blocks already have hash as primary key
-- Add unique constraint on slot_no for additional safety
ALTER TABLE blocks
ADD CONSTRAINT IF NOT EXISTS unique_blocks_slot_no
UNIQUE (slot_no);

-- ============================================
-- TX CONSTRAINTS
-- ============================================

-- Transactions already have hash as primary key
-- Add unique constraint on block_hash + block_index for safety
ALTER TABLE txes
ADD CONSTRAINT IF NOT EXISTS unique_txes_block_index
UNIQUE (block_hash, block_index);

-- ============================================
-- EPOCH CONSTRAINTS
-- ============================================

-- Epoches already have no as primary key
-- No additional constraint needed

-- ============================================
-- SLOT LEADER CONSTRAINTS
-- ============================================

-- Slot leaders already have hash as primary key
-- No additional constraint needed

-- ============================================
-- STAKE ADDRESS CONSTRAINTS
-- ============================================

-- Stake addresses already have hash_raw as primary key
-- View already has unique index
-- No additional constraint needed

-- ============================================
-- POOL HASH CONSTRAINTS
-- ============================================

-- Pool hashes already have hash_raw as primary key
-- View already has unique index
-- No additional constraint needed