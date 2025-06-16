package models

import (
	"time"
	"gorm.io/gorm"
)

// Block represents a block in the blockchain with hash as primary key
type Block struct {
	Hash           []byte    `gorm:"type:VARBINARY(32);primaryKey"`
	EpochNo        *uint32   `gorm:"type:INT UNSIGNED;index"`
	SlotNo         *uint64   `gorm:"type:BIGINT UNSIGNED;uniqueIndex"`
	EpochSlotNo    *uint32   `gorm:"type:INT UNSIGNED"`
	BlockNo        *uint32   `gorm:"type:INT UNSIGNED;index"`
	PreviousHash   []byte    `gorm:"type:VARBINARY(32);index"`
	SlotLeaderHash []byte    `gorm:"type:VARBINARY(28);not null;index"`
	Size           uint32    `gorm:"type:INT UNSIGNED;not null"`
	Time           time.Time `gorm:"not null;index"`
	TxCount        uint64    `gorm:"type:BIGINT UNSIGNED;not null"`
	ProtoMajor     uint32    `gorm:"type:INT UNSIGNED;not null"`
	ProtoMinor     uint32    `gorm:"type:INT UNSIGNED;not null"`
	VrfKey         *string   `gorm:"type:VARCHAR(64)"`
	OpCert         []byte    `gorm:"type:VARBINARY(1024)"`
	OpCertCounter  *uint64   `gorm:"type:BIGINT UNSIGNED"`
	Era            string    `gorm:"type:VARCHAR(16);not null;default:'Byron';index"`

	// Relationships
	Txes []Tx `gorm:"foreignKey:BlockHash;references:Hash"`
}

func (Block) TableName() string {
	return "blocks"
}

// Tx represents a transaction with hash as primary key
type Tx struct {
	Hash             []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	BlockHash        []byte  `gorm:"type:VARBINARY(32);not null;index;index:idx_block_composite,priority:1"`
	BlockIndex       uint32  `gorm:"type:INT UNSIGNED;not null;index:idx_block_composite,priority:2"`
	OutSum           *int64  `gorm:"type:BIGINT"`
	Fee              *int64  `gorm:"type:BIGINT"`
	Deposit          *int64  `gorm:"type:BIGINT"`
	Size             uint32  `gorm:"type:INT UNSIGNED;not null"`
	InvalidBefore    *int64  `gorm:"type:BIGINT"`
	InvalidHereafter *int64  `gorm:"type:BIGINT"`
	ValidContract    *bool   `gorm:"type:BOOLEAN"`
	ScriptSize       *uint32 `gorm:"type:INT UNSIGNED"`
	TreasuryDonation *int64  `gorm:"type:BIGINT"`

	// Relationships
	Block      Block        `gorm:"foreignKey:BlockHash;references:Hash"`
	TxOuts     []TxOut      `gorm:"foreignKey:TxHash;references:Hash"`
	TxIns      []TxIn       `gorm:"foreignKey:TxInHash;references:Hash"`
	TxMetadata []TxMetadata `gorm:"foreignKey:TxHash;references:Hash"`
}

func (Tx) TableName() string {
	return "txes"
}

// TxOut represents a transaction output with composite primary key
type TxOut struct {
	TxHash            []byte  `gorm:"type:VARBINARY(32);primaryKey;index:idx_tx_outs_lookup,priority:1"`
	Index             uint32  `gorm:"type:INT UNSIGNED;primaryKey;index:idx_tx_outs_lookup,priority:2"`
	// Address is VARCHAR(2048) to handle Byron addresses with full HD wallet derivation paths.
	// Normal addresses: Shelley ~100 chars, Byron ~104-114 chars
	// Edge case: Some early Byron addresses with derivation paths can exceed 1000 chars after CBOR+base58 encoding
	Address           string  `gorm:"type:VARCHAR(2048);not null;index:idx_tx_out_address_value,priority:1,length:40"`
	AddressRaw        []byte  `gorm:"type:VARBINARY(1024);not null"`
	AddressHasScript  bool    `gorm:"not null;default:false"`
	PaymentCred       []byte  `gorm:"type:VARBINARY(32);index"`
	StakeAddressHash  []byte  `gorm:"type:VARBINARY(28);index"`
	Value             uint64  `gorm:"type:BIGINT UNSIGNED;not null;index:idx_tx_out_address_value,priority:2"`
	DataHash          []byte  `gorm:"type:VARBINARY(32)"`
	InlineDatumHash   []byte  `gorm:"type:VARBINARY(32);index"`
	ReferenceScriptHash []byte `gorm:"type:VARBINARY(28);index"`

	// Relationships
	Tx           Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	StakeAddress *StakeAddress `gorm:"foreignKey:StakeAddressHash;references:HashRaw"`
	InlineDatum  *Datum        `gorm:"foreignKey:InlineDatumHash;references:Hash"`
	ReferenceScript *Script    `gorm:"foreignKey:ReferenceScriptHash;references:Hash"`
}

func (TxOut) TableName() string {
	return "tx_outs"
}

// TxIn represents a transaction input
type TxIn struct {
	TxInHash     []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	TxInIndex    uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	TxOutHash    []byte  `gorm:"type:VARBINARY(32);not null;index:idx_tx_ins_spent,priority:1"`
	TxOutIndex   uint32  `gorm:"type:INT UNSIGNED;not null;index:idx_tx_ins_spent,priority:2"`
	RedeemerHash []byte  `gorm:"type:VARBINARY(32);index"`

	// Relationships
	TxIn     Tx        `gorm:"foreignKey:TxInHash;references:Hash"`
	TxOut    TxOut     `gorm:"foreignKey:TxOutHash,TxOutIndex;references:TxHash,Index"`
	Redeemer *Redeemer `gorm:"foreignKey:RedeemerHash;references:Hash"`
}

func (TxIn) TableName() string {
	return "tx_ins"
}


// SlotLeader represents a slot leader with hash as primary key
type SlotLeader struct {
	Hash        []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	PoolHash    []byte  `gorm:"type:VARBINARY(28);index"`
	Description *string `gorm:"type:TEXT"`

	// Relationships
	Pool   *PoolHash `gorm:"foreignKey:PoolHash;references:HashRaw"`
	Blocks []Block   `gorm:"foreignKey:SlotLeaderHash;references:Hash"`
}

func (SlotLeader) TableName() string {
	return "slot_leaders"
}

// Epoch represents an epoch
type Epoch struct {
	No        uint32    `gorm:"type:INT UNSIGNED;primaryKey"`
	StartTime time.Time `gorm:"not null;index"`
	EndTime   time.Time `gorm:"not null;index"`
	SlotCount uint32    `gorm:"type:INT UNSIGNED;not null"`
	TxCount   uint64    `gorm:"type:BIGINT UNSIGNED;not null;default:0"`
	BlkCount  uint32    `gorm:"type:INT UNSIGNED;not null;default:0"`

	// Relationships
	Blocks []Block `gorm:"foreignKey:EpochNo;references:No"`
}

func (Epoch) TableName() string {
	return "epoches"
}

// BeforeCreate hooks - Remove all ID-related logic
func (b *Block) BeforeCreate(tx *gorm.DB) error {
	return nil
}

func (t *Tx) BeforeCreate(tx *gorm.DB) error {
	return nil
}

func (to *TxOut) BeforeCreate(tx *gorm.DB) error {
	return nil
}

func (ti *TxIn) BeforeCreate(tx *gorm.DB) error {
	return nil
}


func (sl *SlotLeader) BeforeCreate(tx *gorm.DB) error {
	return nil
}

func (e *Epoch) BeforeCreate(tx *gorm.DB) error {
	return nil
}