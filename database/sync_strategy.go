package database

import (
	"context"
	"fmt"
	"nectar/models"
	"time"

	"gorm.io/gorm"
)

// GetLastProcessedSlot returns the highest slot number from the blocks table
func GetLastProcessedSlot(db *gorm.DB) (uint64, error) {
	var block models.Block
	err := db.Select("slot_no").Order("slot_no DESC").First(&block).Error
	
	if err == gorm.ErrRecordNotFound {
		// No blocks found, start from genesis
		return 0, nil
	}
	
	if err != nil {
		return 0, fmt.Errorf("failed to query last processed slot: %w", err)
	}
	
	if block.SlotNo == nil {
		return 0, nil
	}
	
	return *block.SlotNo, nil
}

// GetBlockCount returns the total number of blocks in the database
func GetBlockCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&models.Block{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count blocks: %w", err)
	}
	return count, nil
}

// GetTxCount returns the total number of transactions in the database
func GetTxCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&models.Tx{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}
	return count, nil
}

// GetSyncStatistics returns comprehensive sync statistics
type SyncStatistics struct {
	LastProcessedSlot uint64    `json:"last_processed_slot"`
	BlockCount       int64     `json:"block_count"`
	TxCount          int64     `json:"tx_count"`
	TxOutCount       int64     `json:"tx_out_count"`
	TxInCount        int64     `json:"tx_in_count"`
	LastUpdateTime   time.Time `json:"last_update_time"`
	SyncStartTime    time.Time `json:"sync_start_time"`
	ProcessingRate   float64   `json:"processing_rate_blocks_per_sec"`
}

func GetSyncStatistics(db *gorm.DB) (*SyncStatistics, error) {
	stats := &SyncStatistics{}
	
	// Get last processed slot
	lastSlot, err := GetLastProcessedSlot(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get last processed slot: %w", err)
	}
	stats.LastProcessedSlot = lastSlot
	
	// Get block count
	blockCount, err := GetBlockCount(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get block count: %w", err)
	}
	stats.BlockCount = blockCount
	
	// Get transaction count
	txCount, err := GetTxCount(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction count: %w", err)
	}
	stats.TxCount = txCount
	
	// Get TX output count
	var txOutCount int64
	if err := db.Model(&models.TxOut{}).Count(&txOutCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count tx outputs: %w", err)
	}
	stats.TxOutCount = txOutCount
	
	// Get TX input count
	var txInCount int64
	if err := db.Model(&models.TxIn{}).Count(&txInCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count tx inputs: %w", err)
	}
	stats.TxInCount = txInCount
	
	// Get timing information
	var firstBlock, lastBlock models.Block
	
	// Get first block time
	if err := db.Order("slot_no ASC").First(&firstBlock).Error; err == nil {
		stats.SyncStartTime = firstBlock.Time
	}
	
	// Get last block time and calculate processing rate
	if err := db.Order("slot_no DESC").First(&lastBlock).Error; err == nil {
		stats.LastUpdateTime = lastBlock.Time
		
		// Calculate processing rate if we have multiple blocks
		if stats.BlockCount > 1 {
			duration := stats.LastUpdateTime.Sub(stats.SyncStartTime)
			if duration.Seconds() > 0 {
				stats.ProcessingRate = float64(stats.BlockCount) / duration.Seconds()
			}
		}
	}
	
	return stats, nil
}

// GetEpochStatistics returns epoch-specific statistics
type EpochStatistics struct {
	CurrentEpoch    uint32 `json:"current_epoch"`
	BlocksInEpoch   int64  `json:"blocks_in_epoch"`
	TxsInEpoch      int64  `json:"txs_in_epoch"`
	EpochStartSlot  uint64 `json:"epoch_start_slot"`
	EpochProgress   float64 `json:"epoch_progress_percent"`
}

func GetEpochStatistics(db *gorm.DB, epochNo uint32) (*EpochStatistics, error) {
	stats := &EpochStatistics{
		CurrentEpoch: epochNo,
	}
	
	// Count blocks in current epoch
	var blockCount int64
	err := db.Model(&models.Block{}).Where("epoch_no = ?", epochNo).Count(&blockCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count blocks in epoch: %w", err)
	}
	stats.BlocksInEpoch = blockCount
	
	// Count transactions in current epoch
	var txCount int64
	err = db.Table("txes").
		Joins("JOIN blocks ON txes.block_id = blocks.id").
		Where("blocks.epoch_no = ?", epochNo).
		Count(&txCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions in epoch: %w", err)
	}
	stats.TxsInEpoch = txCount
	
	// Get epoch start slot
	var firstBlock models.Block
	err = db.Where("epoch_no = ?", epochNo).Order("slot_no ASC").First(&firstBlock).Error
	if err == nil && firstBlock.SlotNo != nil {
		stats.EpochStartSlot = *firstBlock.SlotNo
	}
	
	return stats, nil
}

// CheckDatabaseHealth performs basic health checks on the database
func CheckDatabaseHealth(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Check if we can connect
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	
	// Check if main tables exist and are accessible
	var count int64
	
	// Check blocks table
	if err := db.Model(&models.Block{}).Count(&count).Error; err != nil {
		return fmt.Errorf("blocks table check failed: %w", err)
	}
	
	// Check txes table
	if err := db.Model(&models.Tx{}).Count(&count).Error; err != nil {
		return fmt.Errorf("txes table check failed: %w", err)
	}
	
	return nil
}

// OptimizeDatabase runs database optimization commands for TiDB
func OptimizeDatabase(db *gorm.DB) error {
	// Update table statistics for better query planning
	tables := []string{"blocks", "txes", "tx_outs", "tx_ins", "slot_leaders", "epochs"}
	
	for _, table := range tables {
		query := fmt.Sprintf("ANALYZE TABLE %s", table)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to analyze table %s: %w", table, err)
		}
	}
	
	return nil
} 