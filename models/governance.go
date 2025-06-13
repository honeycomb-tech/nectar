package models

// Custom ENUM types for governance
type VoterRole string
const (
	VoterRoleConstitutionalCommittee VoterRole = "ConstitutionalCommittee"
	VoterRoleDRep                   VoterRole = "DRep" 
	VoterRoleSPO                    VoterRole = "SPO"
)

type VoteChoice string
const (
	VoteChoiceYes     VoteChoice = "Yes"
	VoteChoiceNo      VoteChoice = "No"
	VoteChoiceAbstain VoteChoice = "Abstain"
)

// VotingProcedure represents CIP-1694 voting procedures
type VotingProcedure struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement"`
	TxID                uint64    `gorm:"not null;index"`
	Index               int32     `gorm:"not null"`
	GovActionProposalID uint64    `gorm:"not null;index"`
	VoterRole           VoterRole `gorm:"type:ENUM('ConstitutionalCommittee','DRep','SPO');not null;index"`
	CommitteeVoter      *uint64   `gorm:"type:BIGINT"`
	DRepVoter           *uint64   `gorm:"type:BIGINT"`
	PoolVoter           *uint64   `gorm:"type:BIGINT"`
	Vote                VoteChoice `gorm:"type:ENUM('Yes','No','Abstain');not null;index"`
	VotingAnchorID      *uint64   `gorm:"type:BIGINT"`
	Invalid             *uint64   `gorm:"type:BIGINT"`

	// Relationships
	Tx                  Tx              `gorm:"foreignKey:TxID"`
	GovActionProposal   GovActionProposal `gorm:"foreignKey:GovActionProposalID"`
	CommitteeVoterHash  *CommitteeHash  `gorm:"foreignKey:CommitteeVoter"`
	DRepVoterHash       *DRepHash       `gorm:"foreignKey:DRepVoter"`
	PoolVoterHash       *PoolHash       `gorm:"foreignKey:PoolVoter"`
	VotingAnchor        *VotingAnchor   `gorm:"foreignKey:VotingAnchorID"`
}

// TableName ensures proper table naming to match database reality
func (VotingProcedure) TableName() string {
	return "voting_procedures"
}

// GovActionProposal represents governance action proposals
type GovActionProposal struct {
	ID                  uint64  `gorm:"primaryKey;autoIncrement"`
	TxID                uint64  `gorm:"not null"`
	Index               int32   `gorm:"not null"`
	PrevGovActionIndex  *int32  `gorm:"type:INT"`
	PrevGovActionTxID   *uint64 `gorm:"type:BIGINT"`
	Type                string  `gorm:"type:VARCHAR(50);not null"`
	Description         string  `gorm:"type:TEXT"`
	Deposit             uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	ReturnedEpoch       *uint32 `gorm:"type:INT UNSIGNED"`
	Expiration          *uint32 `gorm:"type:INT UNSIGNED"`
	VotingAnchorID      *uint64 `gorm:"type:BIGINT"`
	ParamProposalID     *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx               Tx              `gorm:"foreignKey:TxID"`
	PrevGovActionTx  *Tx             `gorm:"foreignKey:PrevGovActionTxID"`
	VotingAnchor     *VotingAnchor   `gorm:"foreignKey:VotingAnchorID"`
	ParamProposal    *ParamProposal  `gorm:"foreignKey:ParamProposalID"`
	VotingProcedures []VotingProcedure `gorm:"foreignKey:GovActionProposalID"`
	TreasuryWithdrawals []TreasuryWithdrawal `gorm:"foreignKey:GovActionProposalID"`
}

// TableName ensures proper table naming to match database reality
func (GovActionProposal) TableName() string {
	return "gov_action_proposals"
}

// ParamProposal represents protocol parameter proposals
type ParamProposal struct {
	ID                    uint64   `gorm:"primaryKey;autoIncrement"`
	EpochNo               uint32   `gorm:"type:INT UNSIGNED;not null;index"`
	Key                   uint32   `gorm:"type:INT UNSIGNED;not null"`
	MinFeeA               *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MinFeeB               *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxBlockSize          *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxTxSize             *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxBhSize             *uint64  `gorm:"type:BIGINT UNSIGNED"`
	KeyDeposit            *uint64  `gorm:"type:BIGINT UNSIGNED"`
	PoolDeposit           *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MaxEpoch              *uint64  `gorm:"type:BIGINT UNSIGNED"`
	OptimalPoolCount      *uint64  `gorm:"type:BIGINT UNSIGNED"`
	Influence             *float64 `gorm:"type:DOUBLE"`
	MonetaryExpandRate    *float64 `gorm:"type:DOUBLE"`
	TreasuryGrowthRate    *float64 `gorm:"type:DOUBLE"`
	Decentralisation      *float64 `gorm:"type:DOUBLE"`
	Entropy               []byte   `gorm:"type:VARBINARY(32)"`
	ProtocolMajor         *uint32  `gorm:"type:INT UNSIGNED"`
	ProtocolMinor         *uint32  `gorm:"type:INT UNSIGNED"`
	MinUtxoValue          *uint64  `gorm:"type:BIGINT UNSIGNED"`
	MinPoolCost           *uint64  `gorm:"type:BIGINT UNSIGNED"`
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
	CoinsPerUtxoWord      *uint64  `gorm:"type:BIGINT UNSIGNED"`

	// Relationships
	// Note: EpochParam relationship removed to fix foreign key constraint issues
	// EpochParams should exist independently, not be constrained by ParamProposals
	CostModel         *CostModel     `gorm:"foreignKey:CostModelID"`
	GovActionProposals []GovActionProposal `gorm:"foreignKey:ParamProposalID"`
}

// TableName ensures proper table naming to match database reality
func (ParamProposal) TableName() string {
	return "param_proposals"
}

// DRepHash represents DRep hashes
type DRepHash struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	Raw      []byte `gorm:"type:VARBINARY(28);not null;uniqueIndex"`
	View     string `gorm:"type:VARCHAR(56);not null;uniqueIndex"`
	HasScript bool  `gorm:"not null"`

	// Relationships
	VotingProcedures     []VotingProcedure     `gorm:"foreignKey:DRepVoter"`
	DelegationVotes      []DelegationVote      `gorm:"foreignKey:DRepHashID"`
	DRepDistrs           []DRepDistr           `gorm:"foreignKey:HashID"`
}

// TableName ensures proper table naming to match database reality
func (DRepHash) TableName() string {
	return "d_rep_hashes"
}

// CommitteeHash represents committee member hashes  
type CommitteeHash struct {
	ID  uint64 `gorm:"primaryKey;autoIncrement"`
	Raw []byte `gorm:"type:VARBINARY(28);not null;uniqueIndex"`

	// Relationships
	VotingProcedures         []VotingProcedure         `gorm:"foreignKey:CommitteeVoter"`
	CommitteeRegistrations   []CommitteeRegistration   `gorm:"foreignKey:ColdKeyID"`
	CommitteeRegistrationsHot []CommitteeRegistration  `gorm:"foreignKey:HotKeyID"`
	CommitteeDeregistrations []CommitteeDeregistration `gorm:"foreignKey:ColdKeyID"`
	CommitteeMembers         []CommitteeMember         `gorm:"foreignKey:CommitteeHashID"`
}

// TableName ensures proper table naming to match database reality
func (CommitteeHash) TableName() string {
	return "committee_hashes"
}

// VotingAnchor represents voting anchor information
type VotingAnchor struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	URL      string `gorm:"type:VARCHAR(128);not null"`
	DataHash []byte `gorm:"type:VARBINARY(32);not null"`

	// Relationships
	VotingProcedures   []VotingProcedure   `gorm:"foreignKey:VotingAnchorID"`
	GovActionProposals []GovActionProposal `gorm:"foreignKey:VotingAnchorID"`
}

// TableName ensures proper table naming to match database reality
func (VotingAnchor) TableName() string {
	return "voting_anchors"
}

// DelegationVote represents delegation votes
type DelegationVote struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	TxID       uint64 `gorm:"not null"`
	CertIndex  int32  `gorm:"not null"`
	AddrID     uint64 `gorm:"not null"`
	DRepHashID uint64 `gorm:"not null"`
	RedeemerID *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx       Tx           `gorm:"foreignKey:TxID"`
	Addr     StakeAddress `gorm:"foreignKey:AddrID"`
	DRepHash DRepHash     `gorm:"foreignKey:DRepHashID"`
	Redeemer *Redeemer    `gorm:"foreignKey:RedeemerID"`
}

// TableName ensures proper table naming to match database reality
func (DelegationVote) TableName() string {
	return "delegation_votes"
}

// CommitteeRegistration represents committee registrations
type CommitteeRegistration struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	TxID      uint64 `gorm:"not null"`
	CertIndex int32  `gorm:"not null"`
	ColdKeyID uint64 `gorm:"not null"`
	HotKeyID  uint64 `gorm:"not null"`

	// Relationships
	Tx      Tx            `gorm:"foreignKey:TxID"`
	ColdKey CommitteeHash `gorm:"foreignKey:ColdKeyID"`
	HotKey  CommitteeHash `gorm:"foreignKey:HotKeyID"`
}

// TableName ensures proper table naming to match database reality
func (CommitteeRegistration) TableName() string {
	return "committee_registrations"
}

// CommitteeDeregistration represents committee deregistrations
type CommitteeDeregistration struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	TxID         uint64 `gorm:"not null"`
	CertIndex    int32  `gorm:"not null"`
	ColdKeyID    uint64 `gorm:"not null"`
	AnchorID     *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Tx       Tx            `gorm:"foreignKey:TxID"`
	ColdKey  CommitteeHash `gorm:"foreignKey:ColdKeyID"`
	Anchor   *VotingAnchor `gorm:"foreignKey:AnchorID"`
}

// TableName ensures proper table naming to match database reality
func (CommitteeDeregistration) TableName() string {
	return "committee_deregistrations"
}

// TreasuryWithdrawal represents treasury withdrawals
type TreasuryWithdrawal struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement"`
	GovActionProposalID uint64 `gorm:"not null"`
	StakeAddressID      uint64 `gorm:"not null"`
	Amount              uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	GovActionProposal GovActionProposal `gorm:"foreignKey:GovActionProposalID"`
	StakeAddress      StakeAddress      `gorm:"foreignKey:StakeAddressID"`
}

// TableName ensures proper table naming to match database reality
func (TreasuryWithdrawal) TableName() string {
	return "treasury_withdrawals"
}

// CommitteeMember represents committee members
type CommitteeMember struct {
	ID               uint64 `gorm:"primaryKey;autoIncrement"`
	CommitteeHashID  uint64 `gorm:"not null"`
	FromEpoch        uint32 `gorm:"type:INT UNSIGNED;not null"`
	UntilEpoch       *uint32 `gorm:"type:INT UNSIGNED"`

	// Relationships
	CommitteeHash CommitteeHash `gorm:"foreignKey:CommitteeHashID"`
}

// TableName ensures proper table naming to match database reality
func (CommitteeMember) TableName() string {
	return "committee_members"
}

// EpochState represents epoch state information
type EpochState struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	EpochNo        uint32 `gorm:"type:INT UNSIGNED;not null;uniqueIndex"`
	CommitteeID    *uint64 `gorm:"type:BIGINT"`
	NoConfidenceID *uint64 `gorm:"type:BIGINT"`
	ConstitutionID *uint64 `gorm:"type:BIGINT"`

	// Relationships
	Committee     *Committee         `gorm:"foreignKey:CommitteeID"`
	NoConfidence  *GovActionProposal `gorm:"foreignKey:NoConfidenceID"`
	Constitution  *Constitution      `gorm:"foreignKey:ConstitutionID"`
}

// TableName ensures proper table naming to match database reality
func (EpochState) TableName() string {
	return "epoch_states"
}

// DRepDistr represents DRep distribution
type DRepDistr struct {
	ID     uint64 `gorm:"primaryKey;autoIncrement"`
	HashID uint64 `gorm:"not null"`
	Amount uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	EpochNo uint32 `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Hash DRepHash `gorm:"foreignKey:HashID"`
}

// TableName ensures proper table naming to match database reality
func (DRepDistr) TableName() string {
	return "d_rep_distrs"
}

// Committee represents committee information
type Committee struct {
	ID      uint64 `gorm:"primaryKey;autoIncrement"`
	GovActionProposalID uint64 `gorm:"not null"`
	Quorum  float64 `gorm:"not null"`

	// Relationships
	GovActionProposal GovActionProposal `gorm:"foreignKey:GovActionProposalID"`
	EpochStates       []EpochState      `gorm:"foreignKey:CommitteeID"`
}

// TableName ensures proper table naming to match database reality
func (Committee) TableName() string {
	return "committees"
}

// Constitution represents constitution information
type Constitution struct {
	ID               uint64 `gorm:"primaryKey;autoIncrement"`
	GovActionProposalID uint64 `gorm:"not null"`
	VotingAnchorID   *uint64 `gorm:"type:BIGINT"`
	ScriptHash       []byte  `gorm:"type:VARBINARY(28)"`

	// Relationships
	GovActionProposal GovActionProposal `gorm:"foreignKey:GovActionProposalID"`
	VotingAnchor      *VotingAnchor     `gorm:"foreignKey:VotingAnchorID"`
	EpochStates       []EpochState      `gorm:"foreignKey:ConstitutionID"`
}

// TableName ensures proper table naming to match database reality
func (Constitution) TableName() string {
	return "constitutions"
}

// Treasury represents treasury information
type Treasury struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	TxID            uint64 `gorm:"not null"`
	CertIndex       int32  `gorm:"not null"`
	Amount          uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	StakeAddressID  uint64 `gorm:"not null"`

	// Relationships
	Tx           Tx           `gorm:"foreignKey:TxID"`
	StakeAddress StakeAddress `gorm:"foreignKey:StakeAddressID"`
}

// TableName ensures proper table naming to match database reality
func (Treasury) TableName() string {
	return "treasuries"
}

// Reserve represents reserve information
type Reserve struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	TxID            uint64 `gorm:"not null"`
	CertIndex       int32  `gorm:"not null"`
	Amount          uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	StakeAddressID  uint64 `gorm:"not null"`

	// Relationships
	Tx           Tx           `gorm:"foreignKey:TxID"`
	StakeAddress StakeAddress `gorm:"foreignKey:StakeAddressID"`
}

// TableName ensures proper table naming to match database reality
func (Reserve) TableName() string {
	return "reserves"
}

// PotTransfer represents pot transfers
type PotTransfer struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	TxID      uint64 `gorm:"not null"`
	CertIndex int32  `gorm:"not null"`
	Amount    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxID"`
}

// TableName ensures proper table naming to match database reality
func (PotTransfer) TableName() string {
	return "pot_transfers"
}

// DrepInfo represents Delegated Representative information
type DrepInfo struct {
	ID              uint64  `gorm:"primaryKey;autoIncrement"`
	View            string  `gorm:"type:VARCHAR(255);not null;uniqueIndex"`
	Hash            []byte  `gorm:"type:VARBINARY(32);not null"`
	HasScript       bool    `gorm:"not null"`
	RegisteredTxID  *uint64 `gorm:"type:BIGINT"`
	DeregisteredTxID *uint64 `gorm:"type:BIGINT"`
	VotingAnchorID  *uint64 `gorm:"type:BIGINT"`
	Deposit         *uint64 `gorm:"type:BIGINT UNSIGNED"`

	// Relationships
	RegisteredTx    *Tx          `gorm:"foreignKey:RegisteredTxID"`
	DeregisteredTx  *Tx          `gorm:"foreignKey:DeregisteredTxID"`
	VotingAnchor    *VotingAnchor `gorm:"foreignKey:VotingAnchorID"`
	DelegationVotes []DelegationVote `gorm:"foreignKey:DrepHashID"`
	VotingProcedures []VotingProcedure `gorm:"foreignKey:DrepVoter"`
}

// TableName ensures proper table naming to match database reality
func (DrepInfo) TableName() string {
	return "drep_infos"
}

