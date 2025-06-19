package models

import (
	"crypto/sha256"
	"gorm.io/gorm"
)

// MultiAsset represents native assets/tokens with composite hash key
type MultiAsset struct {
	Policy      []byte `gorm:"type:VARBINARY(28);not null;primaryKey;index:idx_multi_asset_pk,priority:1"`
	Name        []byte `gorm:"type:VARBINARY(32);not null;primaryKey;index:idx_multi_asset_pk,priority:2"`
	Fingerprint string `gorm:"type:VARCHAR(44);not null;uniqueIndex"`

	// Relationships
	MaTxOuts  []MaTxOut  `gorm:"foreignKey:Policy,Name;references:Policy,Name"`
	MaTxMints []MaTxMint `gorm:"foreignKey:Policy,Name;references:Policy,Name"`
}

func (MultiAsset) TableName() string {
	return "multi_assets"
}

// MaTxOut represents multi-asset transaction outputs
type MaTxOut struct {
	TxHash   []byte `gorm:"type:VARBINARY(32);not null;primaryKey;index:idx_ma_tx_out_pk,priority:1"`
	TxIndex  uint32 `gorm:"type:INT UNSIGNED;not null;primaryKey;index:idx_ma_tx_out_pk,priority:2"`
	Policy   []byte `gorm:"type:VARBINARY(28);not null;primaryKey;index:idx_ma_tx_out_pk,priority:3"`
	Name     []byte `gorm:"type:VARBINARY(32);not null;primaryKey;index:idx_ma_tx_out_pk,priority:4"`
	Quantity uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	MultiAsset MultiAsset `gorm:"foreignKey:Policy,Name;references:Policy,Name"`
	TxOut      TxOut      `gorm:"foreignKey:TxHash,TxIndex;references:TxHash,Index"`
}

func (MaTxOut) TableName() string {
	return "ma_tx_outs"
}

// MaTxMint represents minting/burning of multi-assets
type MaTxMint struct {
	TxHash   []byte `gorm:"type:VARBINARY(32);not null;primaryKey;index:idx_ma_tx_mint_pk,priority:1"`
	Policy   []byte `gorm:"type:VARBINARY(28);not null;primaryKey;index:idx_ma_tx_mint_pk,priority:2"`
	Name     []byte `gorm:"type:VARBINARY(32);not null;primaryKey;index:idx_ma_tx_mint_pk,priority:3"`
	Quantity int64  `gorm:"type:BIGINT;not null"` // Can be negative for burns

	// Relationships
	Tx         Tx         `gorm:"foreignKey:TxHash;references:Hash"`
	MultiAsset MultiAsset `gorm:"foreignKey:Policy,Name;references:Policy,Name"`
}

func (MaTxMint) TableName() string {
	return "ma_tx_mints"
}

// Script represents a script with hash as primary key
type Script struct {
	Hash           []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	TxHash         []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Type           string  `gorm:"type:VARCHAR(16);not null"`
	Json           *string `gorm:"type:TEXT"`
	Bytes          []byte  `gorm:"type:BLOB"`
	SerialisedSize *uint32 `gorm:"type:INT UNSIGNED"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxHash;references:Hash"`
}

func (Script) TableName() string {
	return "scripts"
}

// Datum represents script data with hash as primary key
type Datum struct {
	Hash   []byte `gorm:"type:VARBINARY(32);primaryKey"`
	TxHash []byte `gorm:"type:VARBINARY(32);not null;index"`
	Value  []byte `gorm:"type:MEDIUMBLOB"`
	Bytes  []byte `gorm:"type:MEDIUMBLOB;not null"`

	// Relationships
	Tx     Tx      `gorm:"foreignKey:TxHash;references:Hash"`
	TxOuts []TxOut `gorm:"foreignKey:InlineDatumHash;references:Hash"`
}

func (Datum) TableName() string {
	return "data"
}

// RedeemerData represents redeemer data with hash as primary key
type RedeemerData struct {
	Hash   []byte `gorm:"type:VARBINARY(32);primaryKey"`
	TxHash []byte `gorm:"type:VARBINARY(32);not null;index"`
	Value  []byte `gorm:"type:MEDIUMBLOB"`
	Bytes  []byte `gorm:"type:MEDIUMBLOB;not null"`

	// Relationships
	Tx        Tx         `gorm:"foreignKey:TxHash;references:Hash"`
	Redeemers []Redeemer `gorm:"foreignKey:RedeemerDataHash;references:Hash"`
}

func (RedeemerData) TableName() string {
	return "redeemer_data"
}

// Redeemer represents a redeemer
type Redeemer struct {
	Hash             []byte  `gorm:"type:VARBINARY(32);primaryKey"` // Composite hash of tx+index+purpose
	TxHash           []byte  `gorm:"type:VARBINARY(32);not null;index"`
	UnitMem          uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	UnitSteps        uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	Fee              *uint64 `gorm:"type:BIGINT UNSIGNED"`
	Purpose          string  `gorm:"type:VARCHAR(16);not null"`
	Index            uint32  `gorm:"type:INT UNSIGNED;not null"`
	ScriptHash       []byte  `gorm:"type:VARBINARY(28);index"`
	RedeemerDataHash []byte  `gorm:"type:VARBINARY(32);not null;index"`

	// Relationships
	Tx           Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	Script       *Script       `gorm:"foreignKey:ScriptHash;references:Hash"`
	RedeemerData *RedeemerData `gorm:"foreignKey:RedeemerDataHash;references:Hash"`
}

func (Redeemer) TableName() string {
	return "redeemers"
}

// GenerateRedeemerHash generates a unique hash for a redeemer
func GenerateRedeemerHash(txHash []byte, purpose string, index uint32) []byte {
	h := sha256.New()
	h.Write(txHash)
	h.Write([]byte(purpose))
	h.Write([]byte{byte(index >> 24), byte(index >> 16), byte(index >> 8), byte(index)})
	return h.Sum(nil)
}

// CollateralTxIn represents collateral inputs
type CollateralTxIn struct {
	TxInHash   []byte `gorm:"type:VARBINARY(32);primaryKey"`
	TxInIndex  uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	TxOutHash  []byte `gorm:"type:VARBINARY(32);not null;index"`
	TxOutIndex uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	TxIn  Tx    `gorm:"foreignKey:TxInHash;references:Hash"`
	TxOut TxOut `gorm:"foreignKey:TxOutHash,TxOutIndex;references:TxHash,Index"`
}

func (CollateralTxIn) TableName() string {
	return "collateral_tx_ins"
}

// ReferenceTxIn represents reference inputs
type ReferenceTxIn struct {
	TxInHash   []byte `gorm:"type:VARBINARY(32);primaryKey"`
	TxInIndex  uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	TxOutHash  []byte `gorm:"type:VARBINARY(32);not null;index"`
	TxOutIndex uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	TxIn  Tx    `gorm:"foreignKey:TxInHash;references:Hash"`
	TxOut TxOut `gorm:"foreignKey:TxOutHash,TxOutIndex;references:TxHash,Index"`
}

func (ReferenceTxIn) TableName() string {
	return "reference_tx_ins"
}

// CollateralTxOut represents collateral outputs
type CollateralTxOut struct {
	TxHash              []byte `gorm:"type:VARBINARY(32);primaryKey"`
	Index               uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	Address             string `gorm:"type:VARCHAR(1024);not null"`
	AddressRaw          []byte `gorm:"type:VARBINARY(1024);not null"`
	AddressHasScript    bool   `gorm:"not null;default:false"`
	PaymentCred         []byte `gorm:"type:VARBINARY(32);index"`
	StakeAddressHash    []byte `gorm:"type:VARBINARY(28);index"`
	Value               uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	DataHash            []byte `gorm:"type:VARBINARY(32)"`
	MultiAssetsCount    uint32 `gorm:"type:INT UNSIGNED;not null;default:0"`
	InlineDatumHash     []byte `gorm:"type:VARBINARY(32);index"`
	ReferenceScriptHash []byte `gorm:"type:VARBINARY(28);index"`

	// Relationships
	Tx              Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	StakeAddress    *StakeAddress `gorm:"foreignKey:StakeAddressHash;references:HashRaw"`
	InlineDatum     *Datum        `gorm:"foreignKey:InlineDatumHash;references:Hash"`
	ReferenceScript *Script       `gorm:"foreignKey:ReferenceScriptHash;references:Hash"`
}

func (CollateralTxOut) TableName() string {
	return "collateral_tx_outs"
}

// TxMetadata represents transaction metadata
type TxMetadata struct {
	TxHash []byte `gorm:"type:VARBINARY(32);primaryKey"`
	Key    uint64 `gorm:"type:BIGINT UNSIGNED;primaryKey"`
	Json   []byte `gorm:"type:MEDIUMBLOB"`
	Bytes  []byte `gorm:"type:MEDIUMBLOB;not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxHash;references:Hash"`
}

func (TxMetadata) TableName() string {
	return "tx_metadata"
}

// TxCbor stores raw CBOR data for transactions
type TxCbor struct {
	TxHash []byte `gorm:"type:VARBINARY(32);primaryKey"`
	Bytes  []byte `gorm:"type:MEDIUMBLOB;not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxHash;references:Hash"`
}

func (TxCbor) TableName() string {
	return "tx_cbors"
}

// ExtraKeyWitness represents extra key witnesses
type ExtraKeyWitness struct {
	Hash    []byte `gorm:"type:VARBINARY(32);primaryKey"` // Composite hash
	TxHash  []byte `gorm:"type:VARBINARY(32);not null;index"`
	KeyHash []byte `gorm:"type:VARBINARY(28);not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxHash;references:Hash"`
}

func (ExtraKeyWitness) TableName() string {
	return "extra_key_witnesses"
}

// GenerateExtraKeyWitnessHash generates a unique hash for an extra key witness
func GenerateExtraKeyWitnessHash(txHash []byte, keyHash []byte) []byte {
	h := sha256.New()
	h.Write(txHash)
	h.Write(keyHash)
	return h.Sum(nil)
}

// CostModel represents cost model parameters
type CostModel struct {
	Hash  []byte `gorm:"type:VARBINARY(32);primaryKey"` // Hash of costs JSON
	Costs string `gorm:"type:TEXT;not null"`

	// Relationships
	EpochParams []EpochParam `gorm:"foreignKey:CostModelHash;references:Hash"`
}

func (CostModel) TableName() string {
	return "cost_models"
}

// EpochParam represents epoch parameters
type EpochParam struct {
	EpochNo                    uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	MinFeeA                    uint32  `gorm:"type:INT UNSIGNED;not null"`
	MinFeeB                    uint32  `gorm:"type:INT UNSIGNED;not null"`
	MaxBlockSize               uint32  `gorm:"type:INT UNSIGNED;not null"`
	MaxTxSize                  uint32  `gorm:"type:INT UNSIGNED;not null"`
	MaxBhSize                  uint32  `gorm:"type:INT UNSIGNED;not null"`
	KeyDeposit                 int64   `gorm:"type:BIGINT;not null"`
	PoolDeposit                int64   `gorm:"type:BIGINT;not null"`
	MaxEpoch                   uint32  `gorm:"type:INT UNSIGNED;not null"`
	NOpt                       uint32  `gorm:"type:INT UNSIGNED;not null"`
	A0                         float64 `gorm:"not null"`
	Rho                        float64 `gorm:"not null"`
	Tau                        float64 `gorm:"not null"`
	Decentralisation           float64 `gorm:"not null"`
	ExtraEntropy               *string `gorm:"type:TEXT"`
	ProtocolMajor              uint32  `gorm:"type:INT UNSIGNED;not null"`
	ProtocolMinor              uint32  `gorm:"type:INT UNSIGNED;not null"`
	MinUtxo                    int64   `gorm:"type:BIGINT;not null"`
	MinPoolCost                int64   `gorm:"type:BIGINT;not null"`
	CoinsPerUtxoSize           *int64  `gorm:"type:BIGINT"`
	CostModelHash              []byte  `gorm:"type:VARBINARY(32);index"`
	PriceMem                   *float64
	PriceStep                  *float64
	MaxTxExMem                 *uint64 `gorm:"type:BIGINT UNSIGNED"`
	MaxTxExSteps               *uint64 `gorm:"type:BIGINT UNSIGNED"`
	MaxBlockExMem              *uint64 `gorm:"type:BIGINT UNSIGNED"`
	MaxBlockExSteps            *uint64 `gorm:"type:BIGINT UNSIGNED"`
	MaxValSize                 *uint64 `gorm:"type:BIGINT UNSIGNED"`
	CollateralPercent          *uint32 `gorm:"type:INT UNSIGNED"`
	MaxCollateralInputs        *uint32 `gorm:"type:INT UNSIGNED"`
	PvtMotionNoConfidence      *float64
	PvtCommitteeNormal         *float64
	PvtCommitteeNoConfidence   *float64
	PvtHardForkInitiation      *float64
	PvtPPSecurityGroup         *float64
	DvtMotionNoConfidence      *float64
	DvtCommitteeNormal         *float64
	DvtCommitteeNoConfidence   *float64
	DvtUpdateToConstitution    *float64
	DvtHardForkInitiation      *float64
	DvtPPNetworkGroup          *float64
	DvtPPEconomicGroup         *float64
	DvtPPTechnicalGroup        *float64
	DvtPPGovGroup              *float64
	DvtTreasuryWithdrawal      *float64
	CommitteeMinSize           *uint64 `gorm:"type:BIGINT UNSIGNED"`
	CommitteeMaxTermLength     *uint64 `gorm:"type:BIGINT UNSIGNED"`
	GovActionLifetime          *uint64 `gorm:"type:BIGINT UNSIGNED"`
	GovActionDeposit           *uint64 `gorm:"type:BIGINT UNSIGNED"`
	DrepDeposit                *uint64 `gorm:"type:BIGINT UNSIGNED"`
	DrepActivity               *uint64 `gorm:"type:BIGINT UNSIGNED"`
	MinFeeRefScriptCostPerByte *float64

	// Relationships
	CostModel *CostModel `gorm:"foreignKey:CostModelHash;references:Hash"`
}

func (EpochParam) TableName() string {
	return "epoch_params"
}

// AdaPots represents ADA distribution
type AdaPots struct {
	SlotNo   uint64 `gorm:"type:BIGINT UNSIGNED;primaryKey"`
	EpochNo  uint32 `gorm:"type:INT UNSIGNED;not null"`
	Treasury uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Reserves uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Rewards  uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Utxo     uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Deposits uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Fees     uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
}

func (AdaPots) TableName() string {
	return "ada_pots"
}

// EventInfo stores indexer event information
type EventInfo struct {
	EventID   uint64 `gorm:"type:BIGINT UNSIGNED;primaryKey;autoIncrement"`
	EventType string `gorm:"type:VARCHAR(100);not null;index"`
	Timestamp uint64 `gorm:"type:BIGINT UNSIGNED;not null;index"`
	Data      string `gorm:"type:TEXT"`
}

func (EventInfo) TableName() string {
	return "event_infos"
}

// Remove all BeforeCreate hooks since we don't need ID management
func (m *MultiAsset) BeforeCreate(tx *gorm.DB) error      { return nil }
func (m *MaTxOut) BeforeCreate(tx *gorm.DB) error         { return nil }
func (m *MaTxMint) BeforeCreate(tx *gorm.DB) error        { return nil }
func (s *Script) BeforeCreate(tx *gorm.DB) error          { return nil }
func (d *Datum) BeforeCreate(tx *gorm.DB) error           { return nil }
func (r *RedeemerData) BeforeCreate(tx *gorm.DB) error    { return nil }
func (r *Redeemer) BeforeCreate(tx *gorm.DB) error        { return nil }
func (c *CollateralTxIn) BeforeCreate(tx *gorm.DB) error  { return nil }
func (r *ReferenceTxIn) BeforeCreate(tx *gorm.DB) error   { return nil }
func (c *CollateralTxOut) BeforeCreate(tx *gorm.DB) error { return nil }
func (t *TxMetadata) BeforeCreate(tx *gorm.DB) error      { return nil }
func (t *TxCbor) BeforeCreate(tx *gorm.DB) error          { return nil }
func (e *ExtraKeyWitness) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *CostModel) BeforeCreate(tx *gorm.DB) error       { return nil }
func (e *EpochParam) BeforeCreate(tx *gorm.DB) error      { return nil }
func (a *AdaPots) BeforeCreate(tx *gorm.DB) error         { return nil }
func (e *EventInfo) BeforeCreate(tx *gorm.DB) error       { return nil }
