package models

// StakeAddress represents stake addresses
type StakeAddress struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	HashRaw    []byte `gorm:"type:VARBINARY(29);not null"`
	View       string `gorm:"type:VARCHAR(100);not null;uniqueIndex"`
	ScriptHash []byte `gorm:"type:VARBINARY(28)"`

	// Relationships
	TxOuts           []TxOut           `gorm:"foreignKey:StakeAddressID"`
	StakeRegistrations []StakeRegistration `gorm:"foreignKey:AddrID"`
	StakeDeregistrations []StakeDeregistration `gorm:"foreignKey:AddrID"`
	Delegations      []Delegation      `gorm:"foreignKey:AddrID"`
	Rewards          []Reward          `gorm:"foreignKey:AddrID"`
	Withdrawals      []Withdrawal      `gorm:"foreignKey:AddrID"`
	EpochStakes      []EpochStake      `gorm:"foreignKey:AddrID"`
}

// PoolHash represents stake pool hash information
type PoolHash struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	HashRaw  []byte `gorm:"type:VARBINARY(28);not null;uniqueIndex"`
	View     string `gorm:"type:VARCHAR(56);not null;uniqueIndex"`

	// Relationships
	PoolUpdates       []PoolUpdate       `gorm:"foreignKey:HashID"`
	PoolRetires       []PoolRetire       `gorm:"foreignKey:HashID"`
	Rewards           []Reward           `gorm:"foreignKey:PoolID"`
	Delegations       []Delegation       `gorm:"foreignKey:PoolHashID"`
	VotingProcedures  []VotingProcedure  `gorm:"foreignKey:PoolVoter"`
	EpochStakes       []EpochStake       `gorm:"foreignKey:PoolID"`
	OffChainPoolData  []OffChainPoolData `gorm:"foreignKey:PoolID"`
}

// PoolUpdate represents pool updates
type PoolUpdate struct {
	ID               uint64  `gorm:"primaryKey;autoIncrement"`
	TxID             uint64  `gorm:"not null"`
	HashID           uint64  `gorm:"not null"`
	CertIndex        int32   `gorm:"not null"`
	VrfKeyHash       []byte  `gorm:"type:VARBINARY(32);not null"`
	Pledge           uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	Cost             uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	Margin           float64 `gorm:"not null"`
	FixedCost        uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	RewardAddr       []byte  `gorm:"type:VARBINARY(29);not null"`
	ActiveEpochNo    uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	MetaID           *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx           Tx               `gorm:"foreignKey:TxID"`
	Hash         PoolHash         `gorm:"foreignKey:HashID"`
	Meta         *PoolMetadataRef `gorm:"foreignKey:MetaID"`
	PoolRelays   []PoolRelay      `gorm:"foreignKey:UpdateID"`
	PoolOwners   []PoolOwner      `gorm:"foreignKey:PoolUpdateID"`
}

// PoolRelay represents pool relay information
type PoolRelay struct {
	ID         uint64  `gorm:"primaryKey;autoIncrement"`
	UpdateID   uint64  `gorm:"not null"`
	IPv4       *string `gorm:"type:VARCHAR(15)"`
	IPv6       *string `gorm:"type:VARCHAR(39)"`
	DNSName    *string `gorm:"type:VARCHAR(64)"`
	DNSSrvName *string `gorm:"type:VARCHAR(64)"`
	Port       *uint32 `gorm:"type:INT UNSIGNED"`

	// Relationships
	Update PoolUpdate `gorm:"foreignKey:UpdateID"`
}

// PoolRetire represents pool retirements
type PoolRetire struct {
	ID               uint64 `gorm:"primaryKey;autoIncrement"`
	TxID             uint64 `gorm:"not null"`
	HashID           uint64 `gorm:"not null"`
	CertIndex        int32  `gorm:"not null"`
	AnnouncedTxID    uint64 `gorm:"not null"`
	RetiringEpoch    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	Tx           Tx       `gorm:"foreignKey:TxID"`
	Hash         PoolHash `gorm:"foreignKey:HashID"`
	AnnouncedTx  Tx       `gorm:"foreignKey:AnnouncedTxID"`
}

// PoolOwner represents pool owners
type PoolOwner struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	AddrID       uint64 `gorm:"not null"`
	PoolUpdateID uint64 `gorm:"not null"`
	PoolHashID   uint64 `gorm:"not null"`

	// Relationships
	Addr       StakeAddress `gorm:"foreignKey:AddrID"`
	PoolUpdate PoolUpdate   `gorm:"foreignKey:PoolUpdateID"`
	PoolHash   PoolHash     `gorm:"foreignKey:PoolHashID"`
}

// PoolMetadataRef represents pool metadata references
type PoolMetadataRef struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	PoolID   uint64 `gorm:"not null"`
	URL      string `gorm:"type:VARCHAR(255);not null"`
	Hash     []byte `gorm:"type:VARBINARY(32);not null"`

	// Relationships
	Pool        PoolUpdate           `gorm:"foreignKey:PoolID"`
	OffChainData []OffChainPoolData  `gorm:"foreignKey:PmrID"`
}

// StakeRegistration represents stake registrations
type StakeRegistration struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	TxID      uint64 `gorm:"not null"`
	CertIndex int32  `gorm:"not null"`
	AddrID    uint64 `gorm:"not null"`
	EpochNo   uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Tx   Tx           `gorm:"foreignKey:TxID"`
	Addr StakeAddress `gorm:"foreignKey:AddrID"`
}

// StakeDeregistration represents stake deregistrations
type StakeDeregistration struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	TxID      uint64 `gorm:"not null"`
	CertIndex int32  `gorm:"not null"`
	AddrID    uint64 `gorm:"not null"`
	EpochNo   uint32 `gorm:"type:INT UNSIGNED;not null"`
	Refund    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	Tx   Tx           `gorm:"foreignKey:TxID"`
	Addr StakeAddress `gorm:"foreignKey:AddrID"`
}

// Delegation represents stake delegations
type Delegation struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	TxID         uint64 `gorm:"not null"`
	CertIndex    int32  `gorm:"not null"`
	AddrID       uint64 `gorm:"not null"`
	PoolHashID   uint64 `gorm:"not null"`
	ActiveEpochNo uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	SlotNo       uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	RedeemerID   *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx       Tx           `gorm:"foreignKey:TxID"`
	Addr     StakeAddress `gorm:"foreignKey:AddrID"`
	PoolHash PoolHash     `gorm:"foreignKey:PoolHashID"`
	Redeemer *Redeemer    `gorm:"foreignKey:RedeemerID"`
}

// Reward represents rewards
type Reward struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	AddrID    uint64 `gorm:"not null"`
	Type      string `gorm:"type:VARCHAR(9);not null"`
	Amount    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	EarnedEpoch uint32 `gorm:"type:INT UNSIGNED;not null"`
	SpendableEpoch uint32 `gorm:"type:INT UNSIGNED;not null"`
	PoolID    uint64 `gorm:"not null"`

	// Relationships
	Addr StakeAddress `gorm:"foreignKey:AddrID"`
	Pool PoolHash     `gorm:"foreignKey:PoolID"`
}

// Withdrawal represents withdrawals
type Withdrawal struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	TxID       uint64 `gorm:"not null"`
	AddrID     uint64 `gorm:"not null"`
	Amount     uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	RedeemerID *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx       Tx           `gorm:"foreignKey:TxID"`
	Addr     StakeAddress `gorm:"foreignKey:AddrID"`
	Redeemer *Redeemer    `gorm:"foreignKey:RedeemerID"`
}

// EpochStake represents epoch stake information
type EpochStake struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	AddrID    uint64 `gorm:"not null"`
	PoolID    uint64 `gorm:"not null"`
	Amount    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	EpochNo   uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Addr StakeAddress `gorm:"foreignKey:AddrID"`
	Pool PoolHash     `gorm:"foreignKey:PoolID"`
}

// PoolStat represents pool statistics
type PoolStat struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	PoolHashID uint64 `gorm:"not null"`
	EpochNo    uint32 `gorm:"type:INT UNSIGNED;not null"`
	NumberOfBlocks uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	NumberOfDelegators uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	Stake      uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	VotingPower float64 `gorm:"not null"`

	// Relationships
	PoolHash PoolHash `gorm:"foreignKey:PoolHashID"`
}

// RewardRest represents reward rest
type RewardRest struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	AddrID    uint64 `gorm:"not null"`
	Type      string `gorm:"type:VARCHAR(9);not null"`
	Amount    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	EarnedEpoch uint32 `gorm:"type:INT UNSIGNED;not null"`
	SpendableEpoch uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Addr StakeAddress `gorm:"foreignKey:AddrID"`
}

// EpochStakeProgress represents epoch stake progress
type EpochStakeProgress struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	Completed bool   `gorm:"not null"`

	// No relationships for this simple tracking table
}

// DelistedPool represents delisted pools
type DelistedPool struct {
	ID     uint64 `gorm:"primaryKey;autoIncrement"`
	HashID uint64 `gorm:"not null"`

	// Relationships
	Hash PoolHash `gorm:"foreignKey:HashID"`
}

// ReservedPoolTicker represents reserved pool tickers
type ReservedPoolTicker struct {
	ID   uint64 `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"type:VARCHAR(50);not null;uniqueIndex"`

	// No relationships for this simple table
}