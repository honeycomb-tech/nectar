package processors

import (
	"fmt"
	"log"
	"nectar/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdaPotsCalculator calculates ADA distribution at epoch boundaries
type AdaPotsCalculator struct {
	db *gorm.DB
}

// NewAdaPotsCalculator creates a new calculator
func NewAdaPotsCalculator(db *gorm.DB) *AdaPotsCalculator {
	return &AdaPotsCalculator{db: db}
}

// CalculateAdaPotsForEpoch calculates ADA distribution for a given epoch
func (apc *AdaPotsCalculator) CalculateAdaPotsForEpoch(epochNo uint32, slotNo uint64) error {
	log.Printf("[ADA_POTS] Calculating ADA distribution for epoch %d at slot %d", epochNo, slotNo)
	
	// Total ADA supply is 45 billion
	const TOTAL_ADA_SUPPLY = 45_000_000_000_000_000 // in lovelace
	
	// Calculate circulating supply from UTXO set
	var utxoTotal int64
	err := apc.db.Model(&models.TxOut{}).
		Select("COALESCE(SUM(value), 0)").
		Joins("LEFT JOIN tx_ins ON tx_outs.tx_hash = tx_ins.tx_out_hash AND tx_outs.tx_index = tx_ins.tx_out_index").
		Where("tx_ins.tx_out_hash IS NULL"). // Unspent outputs only
		Scan(&utxoTotal).Error
	
	if err != nil {
		return fmt.Errorf("failed to calculate UTXO total: %w", err)
	}
	
	// Calculate fees collected in this epoch
	var feesTotal int64
	err = apc.db.Model(&models.Tx{}).
		Select("COALESCE(SUM(fee), 0)").
		Joins("JOIN blocks ON txes.block_hash = blocks.hash").
		Where("blocks.epoch_no = ?", epochNo).
		Scan(&feesTotal).Error
	
	if err != nil {
		return fmt.Errorf("failed to calculate fees: %w", err)
	}
	
	// Calculate deposits (from stake registrations and pool registrations)
	var depositsTotal int64
	err = apc.db.Model(&models.Tx{}).
		Select("COALESCE(SUM(deposit), 0)").
		Joins("JOIN blocks ON txes.block_hash = blocks.hash").
		Where("blocks.epoch_no = ?", epochNo).
		Scan(&depositsTotal).Error
	
	if err != nil {
		return fmt.Errorf("failed to calculate deposits: %w", err)
	}
	
	// Calculate total rewards distributed (from withdrawals)
	var rewardsTotal int64
	err = apc.db.Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(amount), 0)").
		Joins("JOIN txes ON withdrawals.tx_hash = txes.hash").
		Joins("JOIN blocks ON txes.block_hash = blocks.hash").
		Where("blocks.epoch_no <= ?", epochNo).
		Scan(&rewardsTotal).Error
	
	if err != nil {
		return fmt.Errorf("failed to calculate rewards: %w", err)
	}
	
	// Estimate treasury and reserves
	// In early epochs, treasury starts at 0 and grows from fees
	// Reserves = Total Supply - Circulating - Treasury
	treasuryEstimate := int64(0)
	if epochNo >= 208 { // Shelley onwards
		// Treasury gets 20% of epoch fees (simplified)
		treasuryEstimate = feesTotal * 20 / 100
	}
	
	// Calculate reserves
	reserves := TOTAL_ADA_SUPPLY - utxoTotal - treasuryEstimate - rewardsTotal
	
	// Create ada_pots record
	adaPots := &models.AdaPots{
		SlotNo:    slotNo,
		EpochNo:   epochNo,
		Treasury:  uint64(treasuryEstimate),
		Reserves:  uint64(reserves),
		Rewards:   uint64(rewardsTotal),
		Utxo:      uint64(utxoTotal),
		Deposits:  uint64(depositsTotal),
		Fees:      uint64(feesTotal),
	}
	
	// Insert or update
	err = apc.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slot_no"}},
		UpdateAll: true,
	}).Create(adaPots).Error
	
	if err != nil {
		return fmt.Errorf("failed to save ada_pots: %w", err)
	}
	
	log.Printf("[ADA_POTS] Epoch %d - UTXO: %d, Treasury: %d, Reserves: %d, Fees: %d", 
		epochNo, utxoTotal, treasuryEstimate, reserves, feesTotal)
	
	return nil
}

// BackfillAdaPots calculates ada_pots for all past epochs
func (apc *AdaPotsCalculator) BackfillAdaPots() error {
	log.Printf("[ADA_POTS] Starting backfill of ada_pots data")
	
	// Get all epoch boundaries
	var epochs []struct {
		EpochNo uint32
		SlotNo  uint64
	}
	
	// For each epoch, get the last slot
	err := apc.db.Raw(`
		SELECT 
			epoch_no,
			MAX(slot_no) as slot_no
		FROM blocks
		WHERE epoch_no >= 208  -- Start from Shelley
		GROUP BY epoch_no
		ORDER BY epoch_no
	`).Scan(&epochs).Error
	
	if err != nil {
		return fmt.Errorf("failed to get epochs: %w", err)
	}
	
	log.Printf("[ADA_POTS] Found %d epochs to process", len(epochs))
	
	// Process each epoch
	for i, epoch := range epochs {
		if err := apc.CalculateAdaPotsForEpoch(epoch.EpochNo, epoch.SlotNo); err != nil {
			log.Printf("[ERROR] Failed to calculate ada_pots for epoch %d: %v", epoch.EpochNo, err)
			continue
		}
		
		if i%10 == 0 {
			log.Printf("[ADA_POTS] Progress: %d/%d epochs processed", i+1, len(epochs))
		}
		
		// Small delay to avoid overloading
		if i%100 == 0 && i > 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	log.Printf("[ADA_POTS] Backfill complete!")
	return nil
}