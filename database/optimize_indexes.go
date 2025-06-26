package database

import (
	"log"
	"gorm.io/gorm"
)

// OptimizeIndexes adds performance-critical indexes
func OptimizeIndexes(db *gorm.DB) error {
	log.Println("[INFO] Adding optimized indexes...")

	// Add descending index on blocks.slot_no for fast latest block queries
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_blocks_slot_no_desc ON blocks (slot_no DESC)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create descending slot_no index: %v", err)
	}

	// Add composite index for blocks ordered by slot_no and hash (for deterministic ordering)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_blocks_slot_hash_desc ON blocks (slot_no DESC, hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create composite slot_no/hash index: %v", err)
	}

	// Add index for epoch boundary queries
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_blocks_epoch_slot ON blocks (epoch_no, slot_no)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create epoch/slot index: %v", err)
	}

	// Add indexes for reward calculation queries
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_pool_stats_epoch ON pool_stats (epoch_no)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create pool_stats epoch index: %v", err)
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_delegations_active ON delegations (active_epoch_no, pool_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create delegations active index: %v", err)
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_stake_deregistrations_epoch ON stake_deregistrations (tx_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create stake_deregistrations index: %v", err)
	}

	// Shelley-specific performance indexes
	log.Println("[INFO] Adding Shelley-era performance indexes...")
	
	// Rewards table - composite indexes for common queries
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_rewards_addr_epoch ON rewards (addr_hash, earned_epoch)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create rewards addr_epoch index: %v", err)
	}
	
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_rewards_spendable_epoch ON rewards (spendable_epoch)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create rewards spendable_epoch index: %v", err)
	}
	
	// Pool updates - transaction tracking
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_pool_updates_tx_hash ON pool_updates (tx_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create pool_updates tx_hash index: %v", err)
	}
	
	// Delegations - transaction tracking
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_delegations_tx_hash ON delegations (tx_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create delegations tx_hash index: %v", err)
	}
	
	// Stake registrations - transaction tracking
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_stake_registrations_tx_hash ON stake_registrations (tx_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create stake_registrations tx_hash index: %v", err)
	}
	
	// Pool retirements
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_pool_retires_pool_hash ON pool_retires (pool_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create pool_retires pool_hash index: %v", err)
	}
	
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_pool_retires_retiring_epoch ON pool_retires (retiring_epoch)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create pool_retires retiring_epoch index: %v", err)
	}
	
	// Pool owners
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_pool_owners_pool_update ON pool_owners (pool_update_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create pool_owners pool_update index: %v", err)
	}
	
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_pool_owners_addr ON pool_owners (addr_hash)`).Error; err != nil {
		log.Printf("[WARNING] Failed to create pool_owners addr index: %v", err)
	}

	log.Println("[OK] Optimized indexes added")
	return nil
}