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

	log.Println("[OK] Optimized indexes added")
	return nil
}