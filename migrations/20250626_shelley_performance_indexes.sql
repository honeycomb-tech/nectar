-- Performance indexes for Shelley era based on actual table structure
-- These prevent slow queries during certificate and reward processing

-- Rewards table - composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_rewards_addr_epoch ON rewards(addr_hash, earned_epoch);
CREATE INDEX IF NOT EXISTS idx_rewards_spendable_epoch ON rewards(spendable_epoch);
CREATE INDEX IF NOT EXISTS idx_rewards_addr_type ON rewards(addr_hash, type);

-- Pool updates - already has active_epoch_no index, add tx tracking
CREATE INDEX IF NOT EXISTS idx_pool_updates_tx_hash ON pool_updates(tx_hash);
CREATE INDEX IF NOT EXISTS idx_pool_updates_pledge ON pool_updates(pledge);
CREATE INDEX IF NOT EXISTS idx_pool_updates_margin ON pool_updates(margin);

-- Delegations - add transaction tracking
CREATE INDEX IF NOT EXISTS idx_delegations_tx_hash ON delegations(tx_hash);

-- Stake registrations - add transaction and epoch tracking  
CREATE INDEX IF NOT EXISTS idx_stake_registrations_tx_hash ON stake_registrations(tx_hash);
CREATE INDEX IF NOT EXISTS idx_stake_registrations_epoch ON stake_registrations(epoch_no);

-- Pool retirements
CREATE INDEX IF NOT EXISTS idx_pool_retires_pool_hash ON pool_retires(pool_hash);
CREATE INDEX IF NOT EXISTS idx_pool_retires_retiring_epoch ON pool_retires(retiring_epoch);

-- Stake deregistrations
CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_addr_hash ON stake_deregistrations(addr_hash);
CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_epoch ON stake_deregistrations(epoch_no);

-- Pool owners
CREATE INDEX IF NOT EXISTS idx_pool_owners_pool_update ON pool_owners(pool_update_hash);
CREATE INDEX IF NOT EXISTS idx_pool_owners_addr ON pool_owners(addr_hash);

-- TiDB specific: Help with distributed joins
CREATE INDEX IF NOT EXISTS idx_blocks_epoch_slot ON blocks(epoch_no, slot_no);
CREATE INDEX IF NOT EXISTS idx_stake_addresses_hash_view ON stake_addresses(hash_raw, view);

-- Update table statistics for query optimizer
ANALYZE TABLE stake_addresses;
ANALYZE TABLE delegations;
ANALYZE TABLE pool_updates;
ANALYZE TABLE rewards;
ANALYZE TABLE stake_registrations;
ANALYZE TABLE pool_retires;
ANALYZE TABLE stake_deregistrations;