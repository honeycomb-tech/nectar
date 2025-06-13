package models

import (
	"fmt"
	"time"
)

// SyncCheckpoint represents a checkpoint in the blockchain sync process
type SyncCheckpoint struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// Blockchain position
	SlotNo    uint64 `gorm:"type:BIGINT UNSIGNED;not null;index:idx_slot_no" json:"slot_no"`
	BlockHash string `gorm:"type:VARCHAR(64);not null" json:"block_hash"`
	BlockID   uint64 `gorm:"type:BIGINT UNSIGNED;not null" json:"block_id"`
	Era       string `gorm:"type:VARCHAR(20);not null;index:idx_era" json:"era"`
	EpochNo   uint32 `gorm:"type:INT UNSIGNED;not null" json:"epoch_no"`

	// Checkpoint metadata
	CheckpointType string    `gorm:"type:ENUM('era_boundary','block_interval','manual');not null;default:'block_interval';index:idx_checkpoint_type" json:"checkpoint_type"`
	CreatedAt      time.Time `gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP;index:idx_created_at" json:"created_at"`
	IsValidated    bool      `gorm:"type:BOOLEAN;default:false;index:idx_validated" json:"is_validated"`

	// Data integrity verification
	IntegrityHash string `gorm:"type:VARCHAR(64);not null" json:"integrity_hash"`
	TableCounts   string `gorm:"type:JSON;not null" json:"table_counts"` // JSON string of table counts

	// Reference points for resume
	LastTxID    uint64 `gorm:"type:BIGINT UNSIGNED;not null" json:"last_tx_id"`
	LastBlockID uint64 `gorm:"type:BIGINT UNSIGNED;not null" json:"last_block_id"`
	LastTxOutID uint64 `gorm:"type:BIGINT UNSIGNED;not null" json:"last_tx_out_id"`

	// Performance stats at checkpoint
	TotalBlocks     uint64  `gorm:"type:BIGINT UNSIGNED;not null" json:"total_blocks"`
	TotalTxs        uint64  `gorm:"type:BIGINT UNSIGNED;not null" json:"total_txs"`
	ProcessingRate  float64 `gorm:"type:DECIMAL(10,2);default:0.00" json:"processing_rate"`

	// Notes for debugging
	Notes string `gorm:"type:TEXT" json:"notes"`

	// Relationships
	ResumeAttempts []CheckpointResumeLog `gorm:"foreignKey:CheckpointID" json:"resume_attempts,omitempty"`
}

// CheckpointResumeLog tracks attempts to resume from checkpoints
type CheckpointResumeLog struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CheckpointID        uint64    `gorm:"type:BIGINT UNSIGNED;not null" json:"checkpoint_id"`
	ResumeAttemptedAt   time.Time `gorm:"type:TIMESTAMP;default:CURRENT_TIMESTAMP;index:idx_attempted_at" json:"resume_attempted_at"`
	ResumeSuccessful    bool      `gorm:"type:BOOLEAN;default:false;index:idx_successful" json:"resume_successful"`
	ErrorMessage        string    `gorm:"type:TEXT" json:"error_message,omitempty"`
	RecoveryTimeSeconds uint32    `gorm:"type:INT UNSIGNED" json:"recovery_time_seconds"`

	// Relationships
	Checkpoint SyncCheckpoint `gorm:"foreignKey:CheckpointID" json:"checkpoint,omitempty"`
}

// TableName ensures proper table naming
func (SyncCheckpoint) TableName() string {
	return "sync_checkpoints"
}

func (CheckpointResumeLog) TableName() string {
	return "checkpoint_resume_log"
}

// CheckpointType constants
const (
	CheckpointTypeEraBoundary   = "era_boundary"
	CheckpointTypeBlockInterval = "block_interval"
	CheckpointTypeManual        = "manual"
)

// Era constants for validation
const (
	EraByron   = "byron"
	EraShelley = "shelley"
	EraAllegra = "allegra"
	EraMary    = "mary"
	EraAlonzo  = "alonzo"
	EraBabbage = "babbage"
	EraConway  = "conway"
)

// CheckpointStats represents parsed table counts from JSON
type CheckpointStats struct {
	Blocks            int64 `json:"blocks"`
	Txes              int64 `json:"txes"`
	TxOuts            int64 `json:"tx_outs"`
	TxIns             int64 `json:"tx_ins"`
	StakeAddresses    int64 `json:"stake_addresses"`
	PoolUpdates       int64 `json:"pool_updates"`
	Delegations       int64 `json:"delegations"`
	StakeRegistrations int64 `json:"stake_registrations"`
	MultiAssets       int64 `json:"multi_assets"`
}

// Helper methods for SyncCheckpoint

// IsEraBoundary returns true if this is an era boundary checkpoint
func (cp *SyncCheckpoint) IsEraBoundary() bool {
	return cp.CheckpointType == CheckpointTypeEraBoundary
}

// IsBlockInterval returns true if this is a block interval checkpoint
func (cp *SyncCheckpoint) IsBlockInterval() bool {
	return cp.CheckpointType == CheckpointTypeBlockInterval
}

// IsManual returns true if this is a manual checkpoint
func (cp *SyncCheckpoint) IsManual() bool {
	return cp.CheckpointType == CheckpointTypeManual
}

// GetAge returns how long ago this checkpoint was created
func (cp *SyncCheckpoint) GetAge() time.Duration {
	return time.Since(cp.CreatedAt)
}

// GetEraProgress returns a descriptive string of the era and progress
func (cp *SyncCheckpoint) GetEraProgress() string {
	return fmt.Sprintf("%s Era - Slot %d (Epoch %d)", cp.Era, cp.SlotNo, cp.EpochNo)
}

// GetProcessingRateString returns a formatted processing rate
func (cp *SyncCheckpoint) GetProcessingRateString() string {
	return fmt.Sprintf("%.2f blocks/sec", cp.ProcessingRate)
}