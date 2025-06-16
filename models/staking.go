package models

import (
	"gorm.io/gorm"
)

// StakeAddress represents stake addresses with hash as primary key
type StakeAddress struct {
	HashRaw    []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	View       string  `gorm:"type:VARCHAR(64);not null;uniqueIndex"`
	ScriptHash []byte  `gorm:"type:VARBINARY(28);index"`

	// Relationships
	TxOuts               []TxOut               `gorm:"foreignKey:StakeAddressHash;references:HashRaw"`
	StakeRegistrations   []StakeRegistration   `gorm:"foreignKey:AddrHash;references:HashRaw"`
	StakeDeregistrations []StakeDeregistration `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Delegations          []Delegation          `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Rewards              []Reward              `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Withdrawals          []Withdrawal          `gorm:"foreignKey:AddrHash;references:HashRaw"`
}

func (StakeAddress) TableName() string {
	return "stake_addresses"
}

// PoolHash represents pool with hash as primary key
type PoolHash struct {
	HashRaw []byte `gorm:"type:VARBINARY(28);primaryKey"`
	View    string `gorm:"type:VARCHAR(64);not null;uniqueIndex"`

	// Relationships
	PoolUpdates      []PoolUpdate      `gorm:"foreignKey:PoolHash;references:HashRaw"`
	PoolRetires      []PoolRetire      `gorm:"foreignKey:PoolHash;references:HashRaw"`
	Rewards          []Reward          `gorm:"foreignKey:PoolHash;references:HashRaw"`
	Delegations      []Delegation      `gorm:"foreignKey:PoolHash;references:HashRaw"`
	VotingProcedures []VotingProcedure `gorm:"foreignKey:PoolVoter;references:HashRaw"`
}

func (PoolHash) TableName() string {
	return "pool_hashes"
}

// PoolUpdate represents pool updates with composite primary key
type PoolUpdate struct {
	TxHash        []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex     uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	PoolHash      []byte  `gorm:"type:VARBINARY(28);not null;index"`
	VrfKeyHash    []byte  `gorm:"type:VARBINARY(32);not null"`
	Pledge        int64   `gorm:"type:BIGINT;not null"`
	Cost          int64   `gorm:"type:BIGINT;not null"`
	Margin        float64 `gorm:"not null"`
	FixedCost     int64   `gorm:"type:BIGINT;not null"`
	RewardAddrHash []byte  `gorm:"type:VARBINARY(28);not null;index"`
	ActiveEpochNo uint32  `gorm:"type:INT UNSIGNED;not null"`
	MetaUrl       *string `gorm:"type:VARCHAR(255)"`
	MetaHash      []byte  `gorm:"type:VARBINARY(32)"`

	// Relationships
	Tx           Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	Pool         PoolHash      `gorm:"foreignKey:PoolHash;references:HashRaw"`
	RewardAddr   StakeAddress  `gorm:"foreignKey:RewardAddrHash;references:HashRaw"`
	PoolRelays   []PoolRelay   `gorm:"foreignKey:UpdateTxHash,UpdateCertIndex;references:TxHash,CertIndex"`
	PoolOwners   []PoolOwner   `gorm:"foreignKey:UpdateTxHash,UpdateCertIndex;references:TxHash,CertIndex"`
}

func (PoolUpdate) TableName() string {
	return "pool_updates"
}

// PoolRelay represents pool relay information
type PoolRelay struct {
	UpdateTxHash    []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	UpdateCertIndex uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	RelayIndex      uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	IpV4            *string `gorm:"type:VARCHAR(15)"`
	IpV6            *string `gorm:"type:VARCHAR(39)"`
	DnsName         *string `gorm:"type:VARCHAR(64)"`
	DnsSrvName      *string `gorm:"type:VARCHAR(64)"`
	Port            *uint32 `gorm:"type:INT UNSIGNED"`

	// Relationships
	PoolUpdate PoolUpdate `gorm:"foreignKey:UpdateTxHash,UpdateCertIndex;references:TxHash,CertIndex"`
}

func (PoolRelay) TableName() string {
	return "pool_relays"
}

// PoolRetire represents pool retirement
type PoolRetire struct {
	TxHash         []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex      uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	PoolHash       []byte `gorm:"type:VARBINARY(28);not null;index"`
	RetiringEpoch  uint32 `gorm:"type:INT UNSIGNED;not null"`
	AnnouncedTxHash []byte `gorm:"type:VARBINARY(32);not null;index"`

	// Relationships
	Tx           Tx       `gorm:"foreignKey:TxHash;references:Hash"`
	Pool         PoolHash `gorm:"foreignKey:PoolHash;references:HashRaw"`
	AnnouncedTx  Tx       `gorm:"foreignKey:AnnouncedTxHash;references:Hash"`
}

func (PoolRetire) TableName() string {
	return "pool_retires"
}

// PoolOwner represents pool owners
type PoolOwner struct {
	UpdateTxHash    []byte `gorm:"type:VARBINARY(32);primaryKey"`
	UpdateCertIndex uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	OwnerHash       []byte `gorm:"type:VARBINARY(28);primaryKey"`

	// Relationships
	PoolUpdate PoolUpdate   `gorm:"foreignKey:UpdateTxHash,UpdateCertIndex;references:TxHash,CertIndex"`
	Owner      StakeAddress `gorm:"foreignKey:OwnerHash;references:HashRaw"`
}

func (PoolOwner) TableName() string {
	return "pool_owners"
}

// PoolMetadataRef represents pool metadata references
type PoolMetadataRef struct {
	Hash        []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	PoolHash    []byte  `gorm:"type:VARBINARY(28);not null;index"`
	Url         string  `gorm:"type:VARCHAR(255);not null"`
	RegisteredTxHash []byte  `gorm:"type:VARBINARY(32);not null;index"`

	// Relationships
	Pool         PoolHash            `gorm:"foreignKey:PoolHash;references:HashRaw"`
	RegisteredTx Tx                  `gorm:"foreignKey:RegisteredTxHash;references:Hash"`
	OffChainData []OffChainPoolData  `gorm:"foreignKey:PmrHash;references:Hash"`
}

func (PoolMetadataRef) TableName() string {
	return "pool_metadata_refs"
}

// StakeRegistration represents stake key registration
type StakeRegistration struct {
	TxHash      []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex   uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	AddrHash    []byte  `gorm:"type:VARBINARY(28);not null;index"`
	RedeemerHash []byte  `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx       Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	Addr     StakeAddress  `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Redeemer *Redeemer     `gorm:"foreignKey:RedeemerHash;references:Hash"`
}

func (StakeRegistration) TableName() string {
	return "stake_registrations"
}

// StakeDeregistration represents stake key deregistration
type StakeDeregistration struct {
	TxHash      []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex   uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	AddrHash    []byte  `gorm:"type:VARBINARY(28);not null;index"`
	RedeemerHash []byte  `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx       Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	Addr     StakeAddress  `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Redeemer *Redeemer     `gorm:"foreignKey:RedeemerHash;references:Hash"`
}

func (StakeDeregistration) TableName() string {
	return "stake_deregistrations"
}

// Delegation represents stake delegation
type Delegation struct {
	TxHash        []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex     uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	AddrHash      []byte  `gorm:"type:VARBINARY(28);not null;index"`
	PoolHash      []byte  `gorm:"type:VARBINARY(28);not null;index"`
	ActiveEpochNo uint32  `gorm:"type:INT UNSIGNED;not null;index"`
	SlotNo        uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	RedeemerHash  []byte  `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx       Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	Addr     StakeAddress  `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Pool     PoolHash      `gorm:"foreignKey:PoolHash;references:HashRaw"`
	Redeemer *Redeemer     `gorm:"foreignKey:RedeemerHash;references:Hash"`
}

func (Delegation) TableName() string {
	return "delegations"
}

// Reward represents staking rewards
type Reward struct {
	AddrHash       []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	Type           string  `gorm:"type:VARCHAR(16);primaryKey"`
	EarnedEpoch    uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	SpendableEpoch uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	Amount         int64   `gorm:"type:BIGINT;not null"`
	PoolHash       []byte  `gorm:"type:VARBINARY(28);index"`

	// Relationships
	Addr StakeAddress `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Pool *PoolHash    `gorm:"foreignKey:PoolHash;references:HashRaw"`
}

func (Reward) TableName() string {
	return "rewards"
}

// Withdrawal represents rewards withdrawal
type Withdrawal struct {
	TxHash       []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	AddrHash     []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	Amount       int64   `gorm:"type:BIGINT;not null"`
	RedeemerHash []byte  `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx       Tx           `gorm:"foreignKey:TxHash;references:Hash"`
	Addr     StakeAddress `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Redeemer *Redeemer    `gorm:"foreignKey:RedeemerHash;references:Hash"`
}

func (Withdrawal) TableName() string {
	return "withdrawals"
}

// EpochStake represents epoch stake distribution
type EpochStake struct {
	EpochNo  uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	AddrHash []byte `gorm:"type:VARBINARY(28);primaryKey"`
	PoolHash []byte `gorm:"type:VARBINARY(28);primaryKey"`
	Amount   int64  `gorm:"type:BIGINT;not null"`

	// Relationships
	Addr StakeAddress `gorm:"foreignKey:AddrHash;references:HashRaw"`
	Pool PoolHash     `gorm:"foreignKey:PoolHash;references:HashRaw"`
}

func (EpochStake) TableName() string {
	return "epoch_stakes"
}

// PoolStat represents pool statistics
type PoolStat struct {
	PoolHash     []byte `gorm:"type:VARBINARY(28);primaryKey"`
	EpochNo      uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	BlkCnt       uint32 `gorm:"type:INT UNSIGNED;not null"`
	DelegatorCnt uint32 `gorm:"type:INT UNSIGNED;not null"`
	Stake        int64  `gorm:"type:BIGINT;not null"`
	Rewards      int64  `gorm:"type:BIGINT;not null"`

	// Relationships
	Pool PoolHash `gorm:"foreignKey:PoolHash;references:HashRaw"`
}

func (PoolStat) TableName() string {
	return "pool_stats"
}

// RewardRest represents unclaimed rewards
type RewardRest struct {
	AddrHash       []byte `gorm:"type:VARBINARY(28);primaryKey"`
	Type           string `gorm:"type:VARCHAR(16);primaryKey"`
	Amount         int64  `gorm:"type:BIGINT;not null"`
	EarnedEpoch    uint32 `gorm:"type:INT UNSIGNED;not null"`
	SpendableEpoch uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Addr StakeAddress `gorm:"foreignKey:AddrHash;references:HashRaw"`
}

func (RewardRest) TableName() string {
	return "reward_rests"
}

// EpochStakeProgress represents stake snapshot progress
type EpochStakeProgress struct {
	EpochNo    uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	Completed  bool   `gorm:"not null;default:false"`
	StartedAt  *uint64 `gorm:"type:BIGINT UNSIGNED"`
	CompletedAt *uint64 `gorm:"type:BIGINT UNSIGNED"`
}

func (EpochStakeProgress) TableName() string {
	return "epoch_stake_progresses"
}

// DelistedPool represents delisted pools
type DelistedPool struct {
	PoolHash []byte `gorm:"type:VARBINARY(28);primaryKey"`

	// Relationships
	Pool PoolHash `gorm:"foreignKey:PoolHash;references:HashRaw"`
}

func (DelistedPool) TableName() string {
	return "delisted_pools"
}

// ReservedPoolTicker represents reserved pool tickers
type ReservedPoolTicker struct {
	Name     string `gorm:"type:VARCHAR(5);primaryKey"`
	PoolHash []byte `gorm:"type:VARBINARY(28);not null;uniqueIndex"`

	// Relationships
	Pool PoolHash `gorm:"foreignKey:PoolHash;references:HashRaw"`
}

func (ReservedPoolTicker) TableName() string {
	return "reserved_pool_tickers"
}

// InstantReward represents instant rewards (MIR)
type InstantReward struct {
	TxHash        []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex     uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	Type          string  `gorm:"type:VARCHAR(16);primaryKey"`
	AddrHash      []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	Amount        int64   `gorm:"type:BIGINT;not null"`
	SpendableEpoch uint32  `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Tx   Tx           `gorm:"foreignKey:TxHash;references:Hash"`
	Addr StakeAddress `gorm:"foreignKey:AddrHash;references:HashRaw"`
}

func (InstantReward) TableName() string {
	return "instant_rewards"
}

// Remove all BeforeCreate hooks since we don't need ID management
func (s *StakeAddress) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolHash) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolUpdate) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolRelay) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolRetire) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolOwner) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolMetadataRef) BeforeCreate(tx *gorm.DB) error { return nil }
func (s *StakeRegistration) BeforeCreate(tx *gorm.DB) error { return nil }
func (s *StakeDeregistration) BeforeCreate(tx *gorm.DB) error { return nil }
func (d *Delegation) BeforeCreate(tx *gorm.DB) error { return nil }
func (r *Reward) BeforeCreate(tx *gorm.DB) error { return nil }
func (w *Withdrawal) BeforeCreate(tx *gorm.DB) error { return nil }
func (e *EpochStake) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PoolStat) BeforeCreate(tx *gorm.DB) error { return nil }
func (r *RewardRest) BeforeCreate(tx *gorm.DB) error { return nil }
func (e *EpochStakeProgress) BeforeCreate(tx *gorm.DB) error { return nil }
func (d *DelistedPool) BeforeCreate(tx *gorm.DB) error { return nil }
func (r *ReservedPoolTicker) BeforeCreate(tx *gorm.DB) error { return nil }
func (i *InstantReward) BeforeCreate(tx *gorm.DB) error { return nil }