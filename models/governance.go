package models

import (
	"crypto/sha256"
	"unsafe"
	"gorm.io/gorm"
)

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

// VotingProcedure represents CIP-1694 voting procedures with composite primary key
type VotingProcedure struct {
	TxHash                []byte     `gorm:"type:VARBINARY(32);primaryKey"`
	Index                 uint32     `gorm:"type:INT UNSIGNED;primaryKey"`
	GovActionProposalHash []byte     `gorm:"type:VARBINARY(32);not null;index"`
	VoterRole             VoterRole  `gorm:"type:VARCHAR(32);not null;index"`
	CommitteeVoter        []byte     `gorm:"type:VARBINARY(28);index"`
	DRepVoter             []byte     `gorm:"type:VARBINARY(28);index"`
	PoolVoter             []byte     `gorm:"type:VARBINARY(28);index"`
	Vote                  VoteChoice `gorm:"type:VARCHAR(16);not null;index"`
	VotingAnchorHash      []byte     `gorm:"type:VARBINARY(32);index"`
	Invalid               *bool      `gorm:"default:false"`

	// Relationships
	Tx                  Tx                 `gorm:"foreignKey:TxHash;references:Hash"`
	GovActionProposal   GovActionProposal  `gorm:"foreignKey:GovActionProposalHash;references:Hash"`
	CommitteeVoterHash  *CommitteeHash     `gorm:"foreignKey:CommitteeVoter;references:HashRaw"`
	DRepVoterHash       *DRepHash          `gorm:"foreignKey:DRepVoter;references:HashRaw"`
	PoolVoterHash       *PoolHash          `gorm:"foreignKey:PoolVoter;references:HashRaw"`
	VotingAnchor        *VotingAnchor      `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
}

func (VotingProcedure) TableName() string {
	return "voting_procedures"
}

// GovActionProposal represents governance action proposals with hash as primary key
type GovActionProposal struct {
	Hash                []byte  `gorm:"type:VARBINARY(32);primaryKey"` // Hash of TxHash+Index
	TxHash              []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Index               uint32  `gorm:"type:INT UNSIGNED;not null"`
	PrevGovActionIndex  *uint32 `gorm:"type:INT UNSIGNED"`
	PrevGovActionTxHash []byte  `gorm:"type:VARBINARY(32);index"`
	Type                string  `gorm:"type:VARCHAR(50);not null;index"`
	Description         string  `gorm:"type:TEXT"`
	Deposit             uint64  `gorm:"type:BIGINT UNSIGNED;not null"`
	ReturnedEpoch       *uint32 `gorm:"type:INT UNSIGNED;index"`
	Expiration          *uint32 `gorm:"type:INT UNSIGNED;index"`
	VotingAnchorHash    []byte  `gorm:"type:VARBINARY(32);index"`
	ParamProposalHash   []byte  `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx                  Tx                   `gorm:"foreignKey:TxHash;references:Hash"`
	PrevGovActionTx     *Tx                  `gorm:"foreignKey:PrevGovActionTxHash;references:Hash"`
	VotingAnchor        *VotingAnchor        `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
	ParamProposal       *ParamProposal       `gorm:"foreignKey:ParamProposalHash;references:Hash"`
	VotingProcedures    []VotingProcedure    `gorm:"foreignKey:GovActionProposalHash;references:Hash"`
	TreasuryWithdrawals []TreasuryWithdrawal `gorm:"foreignKey:GovActionProposalHash;references:Hash"`
}

func (GovActionProposal) TableName() string {
	return "gov_action_proposals"
}

// GenerateGovActionProposalHash generates a unique hash for a governance action proposal
func GenerateGovActionProposalHash(txHash []byte, index uint32) []byte {
	h := sha256.New()
	h.Write(txHash)
	h.Write([]byte{byte(index >> 24), byte(index >> 16), byte(index >> 8), byte(index)})
	return h.Sum(nil)
}

// ParamProposal represents protocol parameter proposals with hash as primary key
type ParamProposal struct {
	Hash                  []byte   `gorm:"type:VARBINARY(32);primaryKey"` // Hash of all params
	EpochNo               uint32   `gorm:"type:INT UNSIGNED;not null;index"`
	Key                   uint32   `gorm:"type:INT UNSIGNED;not null;index"`
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
	CostModelHash         []byte   `gorm:"type:VARBINARY(32);index"`
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
	CostModel          *CostModel           `gorm:"foreignKey:CostModelHash;references:Hash"`
	GovActionProposals []GovActionProposal  `gorm:"foreignKey:ParamProposalHash;references:Hash"`
}

func (ParamProposal) TableName() string {
	return "param_proposals"
}

// DRepHash represents DRep hashes with hash as primary key
type DRepHash struct {
	HashRaw   []byte `gorm:"type:VARBINARY(28);primaryKey"`
	View      string `gorm:"type:VARCHAR(56);not null;uniqueIndex"`
	HasScript bool   `gorm:"not null"`

	// Relationships
	VotingProcedures []VotingProcedure `gorm:"foreignKey:DRepVoter;references:HashRaw"`
	DelegationVotes  []DelegationVote  `gorm:"foreignKey:DRepHash;references:HashRaw"`
	DRepDistrs       []DRepDistr       `gorm:"foreignKey:HashRaw;references:HashRaw"`
}

func (DRepHash) TableName() string {
	return "d_rep_hashes"
}

// CommitteeHash represents committee member hashes with hash as primary key
type CommitteeHash struct {
	HashRaw []byte `gorm:"type:VARBINARY(28);primaryKey"`

	// Relationships
	VotingProcedures          []VotingProcedure         `gorm:"foreignKey:CommitteeVoter;references:HashRaw"`
	CommitteeRegistrations    []CommitteeRegistration   `gorm:"foreignKey:ColdKeyHash;references:HashRaw"`
	CommitteeRegistrationsHot []CommitteeRegistration   `gorm:"foreignKey:HotKeyHash;references:HashRaw"`
	CommitteeDeregistrations  []CommitteeDeregistration `gorm:"foreignKey:ColdKeyHash;references:HashRaw"`
	CommitteeMembers          []CommitteeMember         `gorm:"foreignKey:CommitteeHash;references:HashRaw"`
}

func (CommitteeHash) TableName() string {
	return "committee_hashes"
}

// VotingAnchor represents voting anchor information with hash as primary key
type VotingAnchor struct {
	Hash     []byte `gorm:"type:VARBINARY(32);primaryKey"` // Hash of URL+DataHash
	URL      string `gorm:"type:VARCHAR(128);not null;index"`
	DataHash []byte `gorm:"type:VARBINARY(32);not null;index"`

	// Relationships
	VotingProcedures   []VotingProcedure   `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
	GovActionProposals []GovActionProposal `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
}

func (VotingAnchor) TableName() string {
	return "voting_anchors"
}

// GenerateVotingAnchorHash generates a unique hash for a voting anchor
func GenerateVotingAnchorHash(url string, dataHash []byte) []byte {
	h := sha256.New()
	h.Write([]byte(url))
	h.Write(dataHash)
	return h.Sum(nil)
}

// DelegationVote represents delegation votes with composite primary key
type DelegationVote struct {
	TxHash       []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex    uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	AddrHash     []byte `gorm:"type:VARBINARY(28);not null;index"`
	DRepHash     []byte `gorm:"type:VARBINARY(28);not null;index"`
	RedeemerHash []byte `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx       Tx           `gorm:"foreignKey:TxHash;references:Hash"`
	Addr     StakeAddress `gorm:"foreignKey:AddrHash;references:HashRaw"`
	DRep     DRepHash     `gorm:"foreignKey:DRepHash;references:HashRaw"`
	Redeemer *Redeemer    `gorm:"foreignKey:RedeemerHash;references:Hash"`
}

func (DelegationVote) TableName() string {
	return "delegation_votes"
}

// CommitteeRegistration represents committee registrations with composite primary key
type CommitteeRegistration struct {
	TxHash      []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex   uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	ColdKeyHash []byte `gorm:"type:VARBINARY(28);not null;index"`
	HotKeyHash  []byte `gorm:"type:VARBINARY(28);not null;index"`

	// Relationships
	Tx      Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	ColdKey CommitteeHash `gorm:"foreignKey:ColdKeyHash;references:HashRaw"`
	HotKey  CommitteeHash `gorm:"foreignKey:HotKeyHash;references:HashRaw"`
}

func (CommitteeRegistration) TableName() string {
	return "committee_registrations"
}

// CommitteeDeregistration represents committee deregistrations with composite primary key
type CommitteeDeregistration struct {
	TxHash      []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex   uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	ColdKeyHash []byte `gorm:"type:VARBINARY(28);not null;index"`
	AnchorHash  []byte `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Tx      Tx            `gorm:"foreignKey:TxHash;references:Hash"`
	ColdKey CommitteeHash `gorm:"foreignKey:ColdKeyHash;references:HashRaw"`
	Anchor  *VotingAnchor `gorm:"foreignKey:AnchorHash;references:Hash"`
}

func (CommitteeDeregistration) TableName() string {
	return "committee_deregistrations"
}

// TreasuryWithdrawal represents treasury withdrawals with composite primary key
type TreasuryWithdrawal struct {
	GovActionProposalHash []byte `gorm:"type:VARBINARY(32);primaryKey"`
	StakeAddressHash      []byte `gorm:"type:VARBINARY(28);primaryKey"`
	Amount                uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	GovActionProposal GovActionProposal `gorm:"foreignKey:GovActionProposalHash;references:Hash"`
	StakeAddress      StakeAddress      `gorm:"foreignKey:StakeAddressHash;references:HashRaw"`
}

func (TreasuryWithdrawal) TableName() string {
	return "treasury_withdrawals"
}

// CommitteeMember represents committee members with composite primary key
type CommitteeMember struct {
	CommitteeHash []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	FromEpoch     uint32  `gorm:"type:INT UNSIGNED;primaryKey"`
	UntilEpoch    *uint32 `gorm:"type:INT UNSIGNED;index"`

	// Relationships
	Committee CommitteeHash `gorm:"foreignKey:CommitteeHash;references:HashRaw"`
}

func (CommitteeMember) TableName() string {
	return "committee_members"
}

// EpochState represents epoch state information with epoch as primary key
type EpochState struct {
	EpochNo            uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	CommitteeHash      []byte `gorm:"type:VARBINARY(32);index"`
	NoConfidenceHash   []byte `gorm:"type:VARBINARY(32);index"`
	ConstitutionHash   []byte `gorm:"type:VARBINARY(32);index"`

	// Relationships
	Committee     *Committee         `gorm:"foreignKey:CommitteeHash;references:Hash"`
	NoConfidence  *GovActionProposal `gorm:"foreignKey:NoConfidenceHash;references:Hash"`
	Constitution  *Constitution      `gorm:"foreignKey:ConstitutionHash;references:Hash"`
}

func (EpochState) TableName() string {
	return "epoch_states"
}

// DRepDistr represents DRep distribution with composite primary key
type DRepDistr struct {
	HashRaw []byte `gorm:"type:VARBINARY(28);primaryKey"`
	EpochNo uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	Amount  uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	Hash DRepHash `gorm:"foreignKey:HashRaw;references:HashRaw"`
}

func (DRepDistr) TableName() string {
	return "d_rep_distrs"
}

// Committee represents committee information with hash as primary key
type Committee struct {
	Hash                  []byte  `gorm:"type:VARBINARY(32);primaryKey"` // Hash of proposal hash + quorum
	GovActionProposalHash []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Quorum                float64 `gorm:"not null"`

	// Relationships
	GovActionProposal GovActionProposal `gorm:"foreignKey:GovActionProposalHash;references:Hash"`
	EpochStates       []EpochState      `gorm:"foreignKey:CommitteeHash;references:Hash"`
}

func (Committee) TableName() string {
	return "committees"
}

// GenerateCommitteeHash generates a unique hash for a committee
func GenerateCommitteeHash(govActionProposalHash []byte, quorum float64) []byte {
	h := sha256.New()
	h.Write(govActionProposalHash)
	quorumBytes := make([]byte, 8)
	// Convert float64 to bytes
	*(*float64)(unsafe.Pointer(&quorumBytes[0])) = quorum
	h.Write(quorumBytes)
	return h.Sum(nil)
}

// Constitution represents constitution information with hash as primary key
type Constitution struct {
	Hash                  []byte `gorm:"type:VARBINARY(32);primaryKey"` // Hash of proposal + anchor + script
	GovActionProposalHash []byte `gorm:"type:VARBINARY(32);not null;index"`
	VotingAnchorHash      []byte `gorm:"type:VARBINARY(32);index"`
	ScriptHash            []byte `gorm:"type:VARBINARY(28);index"`

	// Relationships
	GovActionProposal GovActionProposal `gorm:"foreignKey:GovActionProposalHash;references:Hash"`
	VotingAnchor      *VotingAnchor     `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
	EpochStates       []EpochState      `gorm:"foreignKey:ConstitutionHash;references:Hash"`
}

func (Constitution) TableName() string {
	return "constitutions"
}

// GenerateConstitutionHash generates a unique hash for a constitution
func GenerateConstitutionHash(govActionProposalHash []byte, votingAnchorHash []byte, scriptHash []byte) []byte {
	h := sha256.New()
	h.Write(govActionProposalHash)
	if votingAnchorHash != nil {
		h.Write(votingAnchorHash)
	}
	if scriptHash != nil {
		h.Write(scriptHash)
	}
	return h.Sum(nil)
}

// Treasury represents treasury information with composite primary key
type Treasury struct {
	TxHash           []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex        uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	Amount           uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	StakeAddressHash []byte `gorm:"type:VARBINARY(28);not null;index"`

	// Relationships
	Tx           Tx           `gorm:"foreignKey:TxHash;references:Hash"`
	StakeAddress StakeAddress `gorm:"foreignKey:StakeAddressHash;references:HashRaw"`
}

func (Treasury) TableName() string {
	return "treasuries"
}

// Reserve represents reserve information with composite primary key
type Reserve struct {
	TxHash           []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex        uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	Amount           uint64 `gorm:"type:BIGINT UNSIGNED;not null"`
	StakeAddressHash []byte `gorm:"type:VARBINARY(28);not null;index"`

	// Relationships
	Tx           Tx           `gorm:"foreignKey:TxHash;references:Hash"`
	StakeAddress StakeAddress `gorm:"foreignKey:StakeAddressHash;references:HashRaw"`
}

func (Reserve) TableName() string {
	return "reserves"
}

// PotTransfer represents pot transfers with composite primary key
type PotTransfer struct {
	TxHash    []byte `gorm:"type:VARBINARY(32);primaryKey"`
	CertIndex uint32 `gorm:"type:INT UNSIGNED;primaryKey"`
	Amount    uint64 `gorm:"type:BIGINT UNSIGNED;not null"`

	// Relationships
	Tx Tx `gorm:"foreignKey:TxHash;references:Hash"`
}

func (PotTransfer) TableName() string {
	return "pot_transfers"
}

// DrepInfo represents Delegated Representative information with hash as primary key
type DrepInfo struct {
	Hash                   []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	View                   string  `gorm:"type:VARCHAR(255);not null;uniqueIndex"`
	HasScript              bool    `gorm:"not null"`
	RegisteredTxHash       []byte  `gorm:"type:VARBINARY(32);index"`
	DeregisteredTxHash     []byte  `gorm:"type:VARBINARY(32);index"`
	VotingAnchorHash       []byte  `gorm:"type:VARBINARY(32);index"`
	Deposit                *uint64 `gorm:"type:BIGINT UNSIGNED"`

	// Relationships
	RegisteredTx     *Tx               `gorm:"foreignKey:RegisteredTxHash;references:Hash"`
	DeregisteredTx   *Tx               `gorm:"foreignKey:DeregisteredTxHash;references:Hash"`
	VotingAnchor     *VotingAnchor     `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
	DelegationVotes  []DelegationVote  `gorm:"foreignKey:DRepHash;references:Hash"`
	VotingProcedures []VotingProcedure `gorm:"foreignKey:DRepVoter;references:Hash"`
}

func (DrepInfo) TableName() string {
	return "drep_infos"
}

// Remove all BeforeCreate hooks since we don't need ID management
func (v *VotingProcedure) BeforeCreate(tx *gorm.DB) error { return nil }
func (g *GovActionProposal) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *ParamProposal) BeforeCreate(tx *gorm.DB) error { return nil }
func (d *DRepHash) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *CommitteeHash) BeforeCreate(tx *gorm.DB) error { return nil }
func (v *VotingAnchor) BeforeCreate(tx *gorm.DB) error { return nil }
func (d *DelegationVote) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *CommitteeRegistration) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *CommitteeDeregistration) BeforeCreate(tx *gorm.DB) error { return nil }
func (t *TreasuryWithdrawal) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *CommitteeMember) BeforeCreate(tx *gorm.DB) error { return nil }
func (e *EpochState) BeforeCreate(tx *gorm.DB) error { return nil }
func (d *DRepDistr) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *Committee) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *Constitution) BeforeCreate(tx *gorm.DB) error { return nil }
func (t *Treasury) BeforeCreate(tx *gorm.DB) error { return nil }
func (r *Reserve) BeforeCreate(tx *gorm.DB) error { return nil }
func (p *PotTransfer) BeforeCreate(tx *gorm.DB) error { return nil }
func (d *DrepInfo) BeforeCreate(tx *gorm.DB) error { return nil }