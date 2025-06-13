package models

// StakeAddress represents stake addresses
type StakeAddress struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	HashRaw    []byte `gorm:"type:VARBINARY(29);not null;uniqueIndex:idx_stake_addresses_hash_raw"`
	View       string `gorm:"type:VARCHAR(100);not null;uniqueIndex"`
	ScriptHash []byte `gorm:"type:VARBINARY(28)"`

	// Relationships
	TxOuts           []TxOut           `gorm:"foreignKey:StakeAddressID"`
	StakeRegistrations []StakeRegistration `gorm:"foreignKey:AddrID"`
	StakeDeregistrations []StakeDeregistration `gorm:"foreignKey:AddrID"`
	Delegations      []Delegation      `gorm:"foreignKey:AddrID"`
	Rewards          []Reward          `gorm:"foreignKey:AddrID"`
	Withdrawals      []Withdrawal      `gorm:"foreignKey:AddrID"`
	EpochStakes      []EpochStake       `gorm:"foreignKey:AddrID"`
}

// TableName ensures proper table naming to match database reality
func (StakeAddress) TableName() string {
	return "stake_addresses"
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

// TableName ensures proper table naming to match database reality
func (PoolHash) TableName() string {
	return "pool_hashes"
}

// PoolUpdate represents pool updates
type PoolUpdate struct {
	ID               uint64  `gorm:"primaryKey;autoIncrement"`
	TxID             uint64  `gorm:"not null"`
	HashID           uint64  `gorm:"not null;index:idx_pool_updates_hash_id"`
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

// TableName ensures proper table naming to match database reality
func (PoolUpdate) TableName() string {
	return "pool_updates"
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

// TableName ensures proper table naming to match database reality
func (PoolRelay) TableName() string {
	return "pool_relays"
}

// PoolRetire represents pool retirements
type PoolRetire struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	TxID            uint64 `gorm:"not null"`
	HashID          uint64 `gorm:"not null"`
	CertIndex       int32  `gorm:"not null"`
	AnnouncedTxID   uint64 `gorm:"not null"`
	RetiringEpoch   uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	Tx          Tx       `gorm:"foreignKey:TxID"`
	Hash        PoolHash `gorm:"foreignKey:HashID"`
	AnnouncedTx Tx       `gorm:"foreignKey:AnnouncedTxID"`
}

// TableName ensures proper table naming to match database reality
func (PoolRetire) TableName() string {
	return "pool_retires"
}

// PoolOwner represents pool owners
type PoolOwner struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	AddrID       uint64 `gorm:"not null"`
	PoolHashID   uint64 `gorm:"not null"`
	PoolUpdateID uint64 `gorm:"not null"`

	// Relationships
	Addr       StakeAddress `gorm:"foreignKey:AddrID"`
	PoolHash   PoolHash     `gorm:"foreignKey:PoolHashID"`
	PoolUpdate PoolUpdate   `gorm:"foreignKey:PoolUpdateID"`
}

// TableName ensures proper table naming to match database reality
func (PoolOwner) TableName() string {
	return "pool_owners"
}

// PoolMetadataRef represents pool metadata references
type PoolMetadataRef struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	PoolID         uint64 `gorm:"not null"`
	URL            string `gorm:"type:VARCHAR(255);not null"`
	Hash           []byte `gorm:"type:VARBINARY(32);not null"`
	RegisteredTxID uint64 `gorm:"not null"`

	// Relationships
	Pool         PoolHash            `gorm:"foreignKey:PoolID"`
	RegisteredTx Tx                  `gorm:"foreignKey:RegisteredTxID"`
	OffChainData []OffChainPoolData  `gorm:"foreignKey:PmrID"`
}

// TableName ensures proper table naming to match database reality
func (PoolMetadataRef) TableName() string {
	return "pool_metadata_refs"
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

// TableName ensures proper table naming to match database reality
func (StakeDeregistration) TableName() string {
	return "stake_deregistrations"
}

// TableName ensures proper table naming to match database reality
func (StakeRegistration) TableName() string {
	return "stake_registrations"
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
	TxID         uint64 `gorm:"not null;index:idx_delegations_tx_id"`
	CertIndex    int32  `gorm:"not null"`
	AddrID       uint64 `gorm:"not null;index:idx_delegations_addr_id"`
	PoolHashID   uint64 `gorm:"not null;index:idx_delegations_pool_hash_id"`
	ActiveEpochNo uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	SlotNo       uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	RedeemerID   *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx       Tx           `gorm:"foreignKey:TxID"`
	Addr     StakeAddress `gorm:"foreignKey:AddrID"`
	PoolHash     PoolHash     `gorm:"foreignKey:PoolHashID"`
	Redeemer     *Redeemer    `gorm:"foreignKey:RedeemerID"`
}

// TableName ensures proper table naming to match database reality
func (Delegation) TableName() string {
	return "delegations"
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

// TableName ensures proper table naming to match database reality
func (EpochStake) TableName() string {
	return "epoch_stakes"
}

// TableName ensures proper table naming to match database reality
func (Reward) TableName() string {
	return "rewards"
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

// TableName ensures proper table naming to match database reality
func (Withdrawal) TableName() string {
	return "withdrawals"
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
	Pool PoolHash `gorm:"foreignKey:PoolHashID"`
}

// TableName ensures proper table naming to match database reality
func (PoolStat) TableName() string {
	return "pool_stats"
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
	Addr     StakeAddress `gorm:"foreignKey:AddrID"`
}

// TableName ensures proper table naming to match database reality
func (RewardRest) TableName() string {
	return "reward_rests"
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

// TableName ensures proper table naming to match database reality
func (DelistedPool) TableName() string {
	return "delisted_pools"
}

// ReservedPoolTicker represents reserved pool tickers
type ReservedPoolTicker struct {
	ID   uint64 `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"type:VARCHAR(50);not null;uniqueIndex"`

	// No relationships for this aggregate table
}

// TableName ensures proper table naming to match database reality
func (ReservedPoolTicker) TableName() string {
	return "reserved_pool_tickers"
}

// TableName ensures proper table naming to match database reality
func (EpochStakeProgress) TableName() string {
	return "epoch_stake_progresses"
}

// InstantReward represents instant rewards from MIR certificates
type InstantReward struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	TxID            uint64 `gorm:"not null"`
	Type            string `gorm:"type:VARCHAR(50);not null"` // treasury or reserves
	CertIndex       uint32 `gorm:"not null"`
	Amount          uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	AddrID          uint64 `gorm:"not null"`
	SpendableEpoch  uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Tx   Tx           `gorm:"foreignKey:TxID"`
	Addr StakeAddress `gorm:"foreignKey:AddrID"`
}

// TableName ensures proper table naming to match database reality
func (InstantReward) TableName() string {
	return "instant_rewards"
}

