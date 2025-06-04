package models

import "time"

// Block represents blocks in the blockchain
type Block struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	Hash          []byte    `gorm:"type:VARBINARY(32);not null;uniqueIndex"`
	EpochNo       *uint32   `gorm:"type:INT UNSIGNED"`
	SlotNo        *uint64   `gorm:"type:BIGINT UNSIGNED"`
	EpochSlotNo   *uint32   `gorm:"type:INT UNSIGNED"`
	BlockNo       *uint32   `gorm:"type:INT UNSIGNED"`
	PreviousID    *uint64   `gorm:"type:BIGINT"`
	SlotLeaderID  uint64    `gorm:"not null"`
	Size          uint32    `gorm:"type:INT UNSIGNED;not null"`
	Time          time.Time `gorm:"not null"`
	TxCount       uint64    `gorm:"type:BIGINT UNSIGNED;not null"`
	ProtoMajor    uint32    `gorm:"type:INT UNSIGNED;not null"`
	ProtoMinor    uint32    `gorm:"type:INT UNSIGNED;not null"`
	VrfKey        *string   `gorm:"type:VARCHAR(64)"`
	OpCert        *[]byte   `gorm:"type:VARBINARY(1024)"`
	OpCertCounter *uint64   `gorm:"type:BIGINT UNSIGNED"`
	Era           string    `gorm:"type:VARCHAR(16);not null;default:'Byron';index"`

	// Relationships
	SlotLeader SlotLeader `gorm:"foreignKey:SlotLeaderID"`
	Previous   *Block     `gorm:"foreignKey:PreviousID"`
	Txes       []Tx       `gorm:"foreignKey:BlockID"`
}

// SlotLeader represents slot leaders
type SlotLeader struct {
	ID          uint64  `gorm:"primaryKey;autoIncrement"`
	Hash        []byte  `gorm:"type:VARBINARY(64);not null;uniqueIndex"`
	PoolHashID  *uint64 `gorm:"type:BIGINT"`
	Description string  `gorm:"type:TEXT"`

	// Relationships
	PoolHash *PoolHash `gorm:"foreignKey:PoolHashID"`
	Blocks   []Block   `gorm:"foreignKey:SlotLeaderID"`
}

// Epoch represents epochs
type Epoch struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	No        uint32    `gorm:"type:INT UNSIGNED;not null;uniqueIndex"`
	StartTime time.Time `gorm:"not null"`
	EndTime   time.Time `gorm:"not null"`
	SlotCount uint32    `gorm:"type:INT UNSIGNED;not null"`
	TxCount   uint64    `gorm:"type:BIGINT UNSIGNED;not null;default:0"`
	BlkCount  uint32    `gorm:"type:INT UNSIGNED;not null;default:0"`

	// No relationships defined for this aggregate table
}

// Tx represents transactions
type Tx struct {
	ID               uint64  `gorm:"primaryKey;autoIncrement"`
	Hash             []byte  `gorm:"type:VARBINARY(32);not null;uniqueIndex"`
	BlockID          uint64  `gorm:"not null"`
	BlockIndex       uint32  `gorm:"type:INT UNSIGNED;not null"`
	OutSum           *uint64 `gorm:"type:BIGINT UNSIGNED"`
	Fee              *uint64 `gorm:"type:BIGINT UNSIGNED"`
	Deposit          *int64  `gorm:"type:BIGINT"`
	Size             uint32  `gorm:"type:INT UNSIGNED;not null"`
	InvalidBefore    *uint64 `gorm:"type:BIGINT UNSIGNED"`
	InvalidHereafter *uint64 `gorm:"type:BIGINT UNSIGNED"`
	ValidContract    *bool   `gorm:"type:BOOLEAN"`
	ScriptSize       *uint32 `gorm:"type:INT UNSIGNED"`
	TreasuryDonation *uint64 `gorm:"type:BIGINT UNSIGNED"`

	// Relationships
	Block      Block        `gorm:"foreignKey:BlockID"`
	TxOuts     []TxOut      `gorm:"foreignKey:TxID"`
	TxIns      []TxIn       `gorm:"foreignKey:TxInID"`
	TxMetadata []TxMetadata `gorm:"foreignKey:TxID"`
}

// TxOut represents transaction outputs
type TxOut struct {
	ID                uint64  `gorm:"primaryKey;autoIncrement"`
	TxID              uint64  `gorm:"not null"`
	Index             uint16  `gorm:"type:SMALLINT UNSIGNED;not null"`
	Address           string  `gorm:"type:VARCHAR(255);not null"`
	AddressRaw        []byte  `gorm:"type:VARBINARY(57)"`
	AddressHasScript  bool    `gorm:"type:BOOLEAN;not null;default:false"`
	PaymentCred       []byte  `gorm:"type:VARBINARY(28)"`
	StakeAddressID    *uint64 `gorm:"type:BIGINT"`
	Value             uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	DataHash          []byte  `gorm:"type:VARBINARY(32)"`
	InlineDatumID     *uint64 `gorm:"type:BIGINT"`
	ReferenceScriptID *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx              Tx            `gorm:"foreignKey:TxID"`
	StakeAddress    *StakeAddress `gorm:"foreignKey:StakeAddressID"`
	InlineDatum     *Datum        `gorm:"foreignKey:InlineDatumID"`
	ReferenceScript *Script       `gorm:"foreignKey:ReferenceScriptID"`
	TxIns           []TxIn        `gorm:"foreignKey:TxOutID"`
}

// TxIn represents transaction inputs
type TxIn struct {
	ID         uint64  `gorm:"primaryKey;autoIncrement"`
	TxInID     uint64  `gorm:"not null"`
	TxOutID    uint64  `gorm:"not null"`
	TxOutIndex uint16  `gorm:"type:SMALLINT UNSIGNED;not null"`
	RedeemerID *uint64 `gorm:"type:BIGINT"`

	// Relationships
	TxIn     Tx        `gorm:"foreignKey:TxInID"`
	TxOut    TxOut     `gorm:"foreignKey:TxOutID"`
	Redeemer *Redeemer `gorm:"foreignKey:RedeemerID"`
}
