package models

// MultiAsset represents native assets/tokens
type MultiAsset struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	Policy      []byte `gorm:"type:VARBINARY(28);not null"`
	Name        []byte `gorm:"type:VARBINARY(32);not null"`
	Fingerprint string `gorm:"type:VARCHAR(44);not null;uniqueIndex"` // CIP14 fingerprint

	// Relationships
	MaTxOuts  []MaTxOut  `gorm:"foreignKey:IdentID"`
	MaTxMints []MaTxMint `gorm:"foreignKey:IdentID"`
}

// TableName ensures proper table naming to match database reality
func (MultiAsset) TableName() string {
	return "multi_assets"
}

// MaTxOut represents multi-asset transaction outputs
type MaTxOut struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	IdentID  uint64 `gorm:"not null"`
	Quantity uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	TxOutID  uint64 `gorm:"not null;index:idx_ma_tx_outs_tx_out_id"`

	// Relationships
	Ident MultiAsset `gorm:"foreignKey:IdentID"`
	TxOut TxOut      `gorm:"foreignKey:TxOutID"`
}

// TableName ensures proper table naming to match database reality
func (MaTxOut) TableName() string {
	return "ma_tx_outs"
}

// MaTxMint represents multi-asset minting
type MaTxMint struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	IdentID  uint64 `gorm:"not null"`
	Quantity int64  `gorm:"type:BIGINT;not null"` // Can be negative for burning
	TxID     uint64 `gorm:"not null"`

	// Relationships
	Ident MultiAsset `gorm:"foreignKey:IdentID"`
	Tx    Tx         `gorm:"foreignKey:TxID"`
}

// TableName ensures proper table naming to match database reality
func (MaTxMint) TableName() string {
	return "ma_tx_mints"
}

// Script represents Plutus and native scripts
type Script struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	TxID         uint64 `gorm:"not null"`
	Hash         []byte `gorm:"type:VARBINARY(28);not null;uniqueIndex"`
	Type         string `gorm:"type:VARCHAR(20);not null"`
	Json         *string `gorm:"type:TEXT"`
	Bytes        []byte `gorm:"type:LONGBLOB"`
	SerializedSize *uint32 `gorm:"type:INT UNSIGNED"`

	// Relationships
	Tx               Tx               `gorm:"foreignKey:TxID"`
	Redeemers        []Redeemer       `gorm:"foreignKey:ScriptHash"`
	TxOuts           []TxOut          `gorm:"foreignKey:ReferenceScriptID"`
	CollateralTxOuts []CollateralTxOut `gorm:"foreignKey:ReferenceScriptID"`
}

// TableName ensures proper table naming to match database reality
func (Script) TableName() string {
	return "scripts"
}

// Datum represents Plutus datums
type Datum struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement"`
	Hash  []byte `gorm:"type:VARBINARY(32);not null;uniqueIndex"`
	TxID  uint64 `gorm:"not null"`
	Value []byte `gorm:"type:LONGBLOB"`
	Bytes []byte `gorm:"type:LONGBLOB"`

	// Relationships
	Tx               Tx               `gorm:"foreignKey:TxID"`
	TxOuts           []TxOut          `gorm:"foreignKey:InlineDatumID"`
	CollateralTxOuts []CollateralTxOut `gorm:"foreignKey:InlineDatumID"`
}

// TableName ensures proper table naming to match database reality
func (Datum) TableName() string {
	return "data"
}

// RedeemerData represents redeemer data
type RedeemerData struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement"`
	Hash  []byte `gorm:"type:VARBINARY(32);not null;uniqueIndex"`
	TxID  uint64 `gorm:"not null"`
	Value []byte `gorm:"type:LONGBLOB"`
	Bytes []byte `gorm:"type:LONGBLOB"`

	// Relationships
	Tx        Tx        `gorm:"foreignKey:TxID"`
	Redeemers []Redeemer `gorm:"foreignKey:DataID"`
}

// TableName ensures proper table naming to match database reality
func (RedeemerData) TableName() string {
	return "redeemer_data"
}

// Redeemer represents transaction redeemers
type Redeemer struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	TxID       uint64 `gorm:"not null"`
	Purpose    string `gorm:"type:VARCHAR(10);not null"`
	Index      uint32 `gorm:"type:INT UNSIGNED;not null"`
	ScriptHash []byte `gorm:"type:VARBINARY(28)"`
	UnitMem    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	UnitSteps  uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	DataID     *uint64 `gorm:"type:BIGINT"`
	Fee        *uint64 `gorm:"type:BIGINT UNSIGNED"`

	// Relationships
	Tx        Tx            `gorm:"foreignKey:TxID"`
	Data      *RedeemerData `gorm:"foreignKey:DataID"`
	Script    *Script       `gorm:"foreignKey:ScriptHash"`
	TxIns     []TxIn        `gorm:"foreignKey:RedeemerID"`
	Delegations []Delegation `gorm:"foreignKey:RedeemerID"`
	Withdrawals []Withdrawal `gorm:"foreignKey:RedeemerID"`
	DelegationVotes []DelegationVote `gorm:"foreignKey:RedeemerID"`
}

// TableName ensures proper table naming to match database reality
func (Redeemer) TableName() string {
	return "redeemers"
}

// CollateralTxIn represents collateral transaction inputs
type CollateralTxIn struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	TxInID     uint64 `gorm:"not null"`
	TxOutID    uint64 `gorm:"not null"`
	TxOutIndex uint16 `gorm:"type:SMALLINT UNSIGNED;not null"`

	// Relationships
	TxIn  Tx    `gorm:"foreignKey:TxInID"`
	TxOut TxOut `gorm:"foreignKey:TxOutID"`
}

// TableName ensures proper table naming to match database reality
func (ReferenceTxIn) TableName() string {
	return "reference_tx_ins"
}

// TableName ensures proper table naming to match database reality
func (CollateralTxIn) TableName() string {
	return "collateral_tx_ins"
}

// ReferenceTxIn represents reference transaction inputs
type ReferenceTxIn struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	TxInID     uint64 `gorm:"not null"`
	TxOutID    uint64 `gorm:"not null"`
	TxOutIndex uint16 `gorm:"type:SMALLINT UNSIGNED;not null"`

	// Relationships
	TxIn  Tx    `gorm:"foreignKey:TxInID"`
	TxOut TxOut `gorm:"foreignKey:TxOutID"`
}

// CollateralTxOut represents collateral transaction outputs
type CollateralTxOut struct {
	ID                uint64  `gorm:"primaryKey;autoIncrement"`
	TxID              uint64  `gorm:"not null"`
	Index             uint16  `gorm:"type:SMALLINT UNSIGNED;not null"`
	Address           string  `gorm:"type:VARCHAR(100);not null"`
	AddressHasScript  bool    `gorm:"not null"`
	PaymentCred       []byte  `gorm:"type:VARBINARY(28)"`
	StakeAddressID    *uint64 `gorm:"type:BIGINT"`
	Value             uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	DataHash          []byte  `gorm:"type:VARBINARY(32)"`
	MultiAssetsDescr  string  `gorm:"type:TEXT;not null"`
	InlineDatumID     *uint64 `gorm:"type:BIGINT"`
	ReferenceScriptID *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx               Tx            `gorm:"foreignKey:TxID"`
	StakeAddress     *StakeAddress `gorm:"foreignKey:StakeAddressID"`
	InlineDatum      *Datum        `gorm:"foreignKey:InlineDatumID"`
	ReferenceScript  *Script       `gorm:"foreignKey:ReferenceScriptID"`
}

// TableName ensures proper table naming to match database reality
func (CollateralTxOut) TableName() string {
	return "collateral_tx_outs"
}

// TxMetadata represents transaction metadata
type TxMetadata struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement"`
	Key   uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Json  *string `gorm:"type:TEXT"`
	Bytes []byte `gorm:"type:LONGBLOB"`
	TxID  uint64 `gorm:"not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxID"`
}

// TableName ensures proper table naming to match database reality
func (TxCbor) TableName() string {
	return "tx_cbors"
}

// TableName ensures proper table naming to match database reality
func (ExtraKeyWitness) TableName() string {
	return "extra_key_witnesses"
}

// TableName ensures proper table naming to match database reality
func (TxMetadata) TableName() string {
	return "tx_metadata"
}

// ExtraKeyWitness represents extra key witnesses
type ExtraKeyWitness struct {
	ID   uint64 `gorm:"primaryKey;autoIncrement"`
	Hash []byte `gorm:"type:VARBINARY(28);not null"`
	TxID uint64 `gorm:"not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxID"`
}

// TxCbor represents transaction CBOR data
type TxCbor struct {
	ID   uint64 `gorm:"primaryKey;autoIncrement"`
	TxID uint64 `gorm:"not null;uniqueIndex"`
	Bytes []byte `gorm:"type:LONGBLOB;not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxID"`
}

// CostModel represents Plutus cost models
type CostModel struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement"`
	Costs string `gorm:"type:TEXT;not null"`
	Hash  []byte `gorm:"type:VARBINARY(32);not null;uniqueIndex"`

	// Relationships
	ParamProposals []ParamProposal `gorm:"foreignKey:CostModelID"`
}

// TableName ensures proper table naming to match database reality
func (CostModel) TableName() string {
	return "cost_models"
}

// EpochParam represents epoch parameters
type EpochParam struct {
	ID                    uint64   `gorm:"primaryKey;autoIncrement"`
	EpochNo               uint32   `gorm:"type:INT UNSIGNED;not null;uniqueIndex"`
	MinFeeA               uint32   `gorm:"type:INT UNSIGNED;not null"`
	MinFeeB               uint32   `gorm:"type:INT UNSIGNED;not null"`
	MaxBlockSize          uint32   `gorm:"type:INT UNSIGNED;not null"`
	MaxTxSize             uint32   `gorm:"type:INT UNSIGNED;not null"`
	MaxBhSize             uint32   `gorm:"type:INT UNSIGNED;not null"`
	KeyDeposit            uint64   `gorm:"type:BIGINT UNSIGNED;not null"`
	PoolDeposit           uint64   `gorm:"type:BIGINT UNSIGNED;not null"`
	MaxEpoch              uint32   `gorm:"type:INT UNSIGNED;not null"`
	OptimalPoolCount      uint32   `gorm:"type:INT UNSIGNED;not null"`
	Influence             float64  `gorm:"not null"`
	MonetaryExpandRate    float64  `gorm:"not null"`
	TreasuryGrowthRate    float64  `gorm:"not null"`
	Decentralisation      float64  `gorm:"not null"`
	Entropy               []byte   `gorm:"type:VARBINARY(32)"`
	ProtocolMajor         uint32   `gorm:"type:INT UNSIGNED;not null"`
	ProtocolMinor         uint32   `gorm:"type:INT UNSIGNED;not null"`
	MinUtxoValue          uint64   `gorm:"type:BIGINT UNSIGNED;not null"`
	MinPoolCost           uint64   `gorm:"type:BIGINT UNSIGNED;not null"`
	Nonce                 []byte   `gorm:"type:VARBINARY(32)"`
	CostModelID           *uint64  `gorm:"type:BIGINT"`
	PriceMem              *float64 `gorm:"type:DOUBLE"`
	PriceStep             *float64 `gorm:"type:DOUBLE"`
	MaxTxExMem            *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxTxExSteps          *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxBlockExMem         *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxBlockExSteps       *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxValSize            *uint64  `gorm:"type:BIGINT UNSIGNED"`
	CollateralPercent     *uint32  `gorm:"type:INT UNSIGNED"`
	MaxCollateralInputs   *uint32  `gorm:"type:INT UNSIGNED"`
	CoinsPerUtxoSize      *uint64  `gorm:"type:BIGINT UNSIGNED"`

	// Relationships
	CostModel *CostModel `gorm:"foreignKey:CostModelID"`
}

// TableName ensures proper table naming to match database reality
func (EpochParam) TableName() string {
	return "epoch_params"
}

// AdaPots represents ADA pots (treasury, reserves, etc.)
type AdaPots struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	SlotNo    uint64 `gorm:"type:BIGINT UNSIGNED;not null;uniqueIndex"`
	EpochNo   uint32 `gorm:"type:INT UNSIGNED;not null"`
	Treasury  uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Reserves  uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Rewards   uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Utxo      uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Deposits  uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Fees      uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// No relationships for this aggregate table
}

// TableName ensures proper table naming to match database reality
func (AdaPots) TableName() string {
	return "ada_pots"
}

// EventInfo represents event information
type EventInfo struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	Type     string `gorm:"type:VARCHAR(50);not null"`
	SlotNo   uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	BlockID  uint64 `gorm:"not null"`
	Details  string `gorm:"type:TEXT"`

	// Relationships
	Block Block `gorm:"foreignKey:BlockID"`
}

// TableName ensures proper table naming to match database reality
func (EventInfo) TableName() string {
	return "event_infos"
}