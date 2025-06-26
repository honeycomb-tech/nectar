package database

import (
	"fmt"
	"log"
	"gorm.io/gorm"
)

// UnifiedIndexManager manages all database indexes in one place
type UnifiedIndexManager struct {
	db *gorm.DB
}

// NewUnifiedIndexManager creates a new index manager
func NewUnifiedIndexManager(db *gorm.DB) *UnifiedIndexManager {
	return &UnifiedIndexManager{db: db}
}

// CreateAllIndexes creates all necessary indexes for optimal performance
func (uim *UnifiedIndexManager) CreateAllIndexes() error {
	log.Println("[INFO] Creating unified index set...")
	
	// Group indexes by category for better organization
	if err := uim.createCoreIndexes(); err != nil {
		return err
	}
	
	if err := uim.createShelleyIndexes(); err != nil {
		return err
	}
	
	if err := uim.createPerformanceIndexes(); err != nil {
		return err
	}
	
	if err := uim.createTiDBOptimizedIndexes(); err != nil {
		return err
	}
	
	log.Println("[OK] All indexes created successfully")
	return nil
}

// createCoreIndexes creates essential indexes for basic operations
func (uim *UnifiedIndexManager) createCoreIndexes() error {
	log.Println("[INFO] Creating core indexes...")
	
	indexes := []string{
		// Blocks - essential for chain traversal
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_no ON blocks(slot_no)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_no_desc ON blocks(slot_no DESC)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_epoch_no ON blocks(epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_block_no ON blocks(block_no)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_block_no_desc ON blocks(block_no DESC)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_time ON blocks(time)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_previous_hash ON blocks(previous_hash)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_leader_hash ON blocks(slot_leader_hash)",
		
		// Transactions - essential for lookups
		"CREATE INDEX IF NOT EXISTS idx_txes_block_hash ON txes(block_hash)",
		"CREATE INDEX IF NOT EXISTS idx_txes_block_index ON txes(block_hash, block_index)",
		
		// Transaction outputs - essential for UTXO queries
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_address ON tx_outs(address(100))",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_stake_address ON tx_outs(stake_address_hash)",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_value ON tx_outs(value)",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_payment_cred ON tx_outs(payment_cred)",
		
		// Transaction inputs - essential for spending
		"CREATE INDEX IF NOT EXISTS idx_tx_ins_spent ON tx_ins(tx_out_hash, tx_out_index)",
		"CREATE INDEX IF NOT EXISTS idx_tx_ins_tx_in ON tx_ins(tx_in_hash)",
		
		// Slot leaders - FIX FOR SLOW QUERY
		"CREATE INDEX IF NOT EXISTS idx_slot_leaders_hash_pool ON slot_leaders(hash, pool_hash)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_slot_leaders_hash ON slot_leaders(hash)",
		
		// Additional performance indexes for common queries
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_address_value ON tx_outs(address(100), value)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_era ON blocks(era)",
	}
	
	return uim.executeIndexes(indexes, "core")
}

// createShelleyIndexes creates indexes for Shelley+ features
func (uim *UnifiedIndexManager) createShelleyIndexes() error {
	log.Println("[INFO] Creating Shelley era indexes...")
	
	indexes := []string{
		// Stake addresses
		"CREATE INDEX IF NOT EXISTS idx_stake_addresses_view ON stake_addresses(view)",
		"CREATE INDEX IF NOT EXISTS idx_stake_addresses_script_hash ON stake_addresses(script_hash)",
		
		// Delegations
		"CREATE INDEX IF NOT EXISTS idx_delegations_addr_hash ON delegations(addr_hash)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_pool_hash ON delegations(pool_hash)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_active_epoch_no ON delegations(active_epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_addr_epoch ON delegations(addr_hash, active_epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_tx_hash ON delegations(tx_hash)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_redeemer_hash ON delegations(redeemer_hash)",
		
		// Pool updates
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_pool_hash ON pool_updates(pool_hash)",
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_active_epoch_no ON pool_updates(active_epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_reward_addr_hash ON pool_updates(reward_addr_hash)",
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_tx_hash ON pool_updates(tx_hash)",
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_pledge ON pool_updates(pledge)",
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_margin ON pool_updates(margin)",
		
		// Pool hashes
		"CREATE INDEX IF NOT EXISTS idx_pool_hashes_view ON pool_hashes(view)",
		
		// Pool owners - using correct column names
		"CREATE INDEX IF NOT EXISTS idx_pool_owners_update_tx ON pool_owners(update_tx_hash)",
		"CREATE INDEX IF NOT EXISTS idx_pool_owners_owner ON pool_owners(owner_hash)",
		
		// Pool retirements
		"CREATE INDEX IF NOT EXISTS idx_pool_retires_pool_hash ON pool_retires(pool_hash)",
		"CREATE INDEX IF NOT EXISTS idx_pool_retires_retiring_epoch ON pool_retires(retiring_epoch)",
		
		// Stake registrations
		"CREATE INDEX IF NOT EXISTS idx_stake_registrations_addr_hash ON stake_registrations(addr_hash)",
		"CREATE INDEX IF NOT EXISTS idx_stake_registrations_tx_hash ON stake_registrations(tx_hash)",
		"CREATE INDEX IF NOT EXISTS idx_stake_registrations_redeemer_hash ON stake_registrations(redeemer_hash)",
		
		// Stake deregistrations
		"CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_addr_hash ON stake_deregistrations(addr_hash)",
		"CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_tx_hash ON stake_deregistrations(tx_hash)",
		
		// Rewards
		"CREATE INDEX IF NOT EXISTS idx_rewards_addr_hash ON rewards(addr_hash)",
		"CREATE INDEX IF NOT EXISTS idx_rewards_pool_hash ON rewards(pool_hash)",
		"CREATE INDEX IF NOT EXISTS idx_rewards_earned_epoch ON rewards(earned_epoch)",
		"CREATE INDEX IF NOT EXISTS idx_rewards_spendable_epoch ON rewards(spendable_epoch)",
		"CREATE INDEX IF NOT EXISTS idx_rewards_addr_epoch ON rewards(addr_hash, earned_epoch)",
		"CREATE INDEX IF NOT EXISTS idx_rewards_type ON rewards(type)",
		
		// Pool stats
		"CREATE INDEX IF NOT EXISTS idx_pool_stats_epoch_hash ON pool_stats(epoch_no, pool_hash)",
		
		// Withdrawals
		"CREATE INDEX IF NOT EXISTS idx_withdrawals_addr_hash ON withdrawals(addr_hash)",
		"CREATE INDEX IF NOT EXISTS idx_withdrawals_tx_hash ON withdrawals(tx_hash)",
	}
	
	return uim.executeIndexes(indexes, "shelley")
}

// createPerformanceIndexes creates indexes for query optimization
func (uim *UnifiedIndexManager) createPerformanceIndexes() error {
	log.Println("[INFO] Creating performance indexes...")
	
	indexes := []string{
		// Composite indexes for common query patterns
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_hash ON blocks(slot_no DESC, hash)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_epoch_slot ON blocks(epoch_no, slot_no)",
		
		// Multi-asset indexes
		"CREATE INDEX IF NOT EXISTS idx_ma_tx_outs_asset ON ma_tx_outs(policy, name)",
		"CREATE INDEX IF NOT EXISTS idx_ma_tx_mints_asset ON ma_tx_mints(policy, name)",
		
		// Certificate indexes - removed as certificates are stored in specific tables
		// "CREATE INDEX IF NOT EXISTS idx_certificates_tx_cert ON certificates(tx_hash, cert_index)",
		// "CREATE INDEX IF NOT EXISTS idx_certificates_type ON certificates(cert_type)",
		
		// Metadata indexes
		"CREATE INDEX IF NOT EXISTS idx_tx_metadata_tx_hash ON tx_metadata(tx_hash)",
		"CREATE INDEX IF NOT EXISTS idx_tx_metadata_key ON tx_metadata(`key`)", // key is a reserved word
		
		// Script indexes
		"CREATE INDEX IF NOT EXISTS idx_scripts_type ON scripts(type)",
		"CREATE INDEX IF NOT EXISTS idx_scripts_tx_hash ON scripts(tx_hash)",
		
		// Redeemer indexes
		"CREATE INDEX IF NOT EXISTS idx_redeemers_tx_hash ON redeemers(tx_hash)",
		"CREATE INDEX IF NOT EXISTS idx_redeemers_purpose ON redeemers(purpose)",
	}
	
	return uim.executeIndexes(indexes, "performance")
}

// createTiDBOptimizedIndexes creates TiDB-specific optimizations
func (uim *UnifiedIndexManager) createTiDBOptimizedIndexes() error {
	log.Println("[INFO] Creating TiDB-optimized indexes...")
	
	indexes := []string{
		// Indexes that help with distributed query execution
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_tx_hash_index ON tx_outs(tx_hash, `index`)",
		"CREATE INDEX IF NOT EXISTS idx_tx_ins_tx_in_hash_index ON tx_ins(tx_in_hash, tx_in_index)",
		
		// Covering indexes for common queries
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_epoch_time ON blocks(slot_no, epoch_no, time)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_pool_epoch ON delegations(pool_hash, active_epoch_no)",
		
		// Hash-based sharding helpers
		"CREATE INDEX IF NOT EXISTS idx_stake_addresses_hash_view ON stake_addresses(hash_raw, view)",
		"CREATE INDEX IF NOT EXISTS idx_pool_hashes_hash_view ON pool_hashes(hash_raw, view)",
	}
	
	return uim.executeIndexes(indexes, "tidb")
}

// executeIndexes runs a list of index creation statements
func (uim *UnifiedIndexManager) executeIndexes(indexes []string, category string) error {
	for _, idx := range indexes {
		if err := uim.db.Exec(idx).Error; err != nil {
			log.Printf("[WARNING] Failed to create %s index: %v (SQL: %s)", category, err, idx)
			// Continue with other indexes even if one fails
		}
	}
	return nil
}

// AnalyzeTables updates table statistics for query optimizer
func (uim *UnifiedIndexManager) AnalyzeTables() error {
	log.Println("[INFO] Analyzing tables for query optimization...")
	
	tables := []string{
		"blocks", "txes", "tx_outs", "tx_ins",
		"stake_addresses", "delegations", "pool_updates", "rewards",
		"slot_leaders", "pool_hashes", "stake_registrations",
		"multi_assets", "ma_tx_outs", "ma_tx_mints",
	}
	
	for _, table := range tables {
		if err := uim.db.Exec(fmt.Sprintf("ANALYZE TABLE %s", table)).Error; err != nil {
			log.Printf("[WARNING] Failed to analyze table %s: %v", table, err)
		}
	}
	
	log.Println("[OK] Table analysis complete")
	return nil
}