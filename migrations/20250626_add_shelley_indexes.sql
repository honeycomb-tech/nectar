-- Additional indexes for Shelley era performance
-- Based on cardano-db-sync schema analysis

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_rewards_addr_epoch ON rewards(addr_hash, earned_epoch);
CREATE INDEX IF NOT EXISTS idx_rewards_spendable_epoch ON rewards(spendable_epoch);
CREATE INDEX IF NOT EXISTS idx_rewards_type ON rewards(type);

-- Pool update indexes
CREATE INDEX IF NOT EXISTS idx_pool_updates_active_epoch ON pool_updates(active_epoch_no);
CREATE INDEX IF NOT EXISTS idx_pool_updates_registered_tx ON pool_updates(registered_tx_hash);

-- Delegation transaction tracking
CREATE INDEX IF NOT EXISTS idx_delegations_tx_hash ON delegations(tx_hash);

-- Stake registration transaction tracking  
CREATE INDEX IF NOT EXISTS idx_stake_registrations_tx_hash ON stake_registrations(tx_hash);
CREATE INDEX IF NOT EXISTS idx_stake_registrations_epoch ON stake_registrations(epoch_no);

-- Pool retirement tracking
CREATE INDEX IF NOT EXISTS idx_pool_retires_pool_hash ON pool_retires(pool_hash);
CREATE INDEX IF NOT EXISTS idx_pool_retires_retiring_epoch ON pool_retires(retiring_epoch);

-- Stake deregistration tracking
CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_addr_hash ON stake_deregistrations(addr_hash);
CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_epoch ON stake_deregistrations(epoch_no);

-- Pool owner tracking
CREATE INDEX IF NOT EXISTS idx_pool_owners_pool_hash ON pool_owners(pool_update_hash);
CREATE INDEX IF NOT EXISTS idx_pool_owners_addr_hash ON pool_owners(addr_hash);

-- TiDB specific optimizations
-- These help with distributed query execution
CREATE INDEX IF NOT EXISTS idx_blocks_epoch_slot ON blocks(epoch_no, slot_no);
CREATE INDEX IF NOT EXISTS idx_txes_block_epoch ON txes(block_hash, block_index);

-- Analyze tables to update statistics
ANALYZE TABLE stake_addresses;
ANALYZE TABLE delegations;
ANALYZE TABLE pool_updates;
ANALYZE TABLE rewards;
ANALYZE TABLE stake_registrations;