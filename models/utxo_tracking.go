package models

import (
	"time"

	"gorm.io/gorm"
)

// UtxoState tracks the UTXO total at each epoch boundary for fast calculation
type UtxoState struct {
	EpochNo         uint32    `gorm:"primaryKey;autoIncrement:false"`
	BlockHash       []byte    `gorm:"type:VARBINARY(32);index"`
	TotalUtxo       uint64    `gorm:"type:DECIMAL(30,0)"`
	CalculationTime time.Time `gorm:"autoCreateTime"`
}

func (UtxoState) TableName() string {
	return "utxo_states"
}

// UtxoDelta tracks UTXO changes per block for incremental calculation
type UtxoDelta struct {
	BlockHash   []byte `gorm:"type:VARBINARY(32);primaryKey"`
	UtxoCreated uint64 `gorm:"type:DECIMAL(20,0);default:0"`
	UtxoSpent   uint64 `gorm:"type:DECIMAL(20,0);default:0"`
	NetChange   int64  `gorm:"->;type:DECIMAL(20,0) GENERATED ALWAYS AS (utxo_created - utxo_spent) STORED"`
}

func (UtxoDelta) TableName() string {
	return "utxo_deltas"
}

// Remove all BeforeCreate hooks since we don't need ID management
func (u *UtxoState) BeforeCreate(tx *gorm.DB) error { return nil }
func (u *UtxoDelta) BeforeCreate(tx *gorm.DB) error { return nil }
