package models

import (
	"crypto/sha256"
	"time"

	"gorm.io/gorm"
)

// OffChainPoolData represents off-chain pool metadata with hash as primary key
type OffChainPoolData struct {
	Hash       []byte `gorm:"type:VARBINARY(32);primaryKey"`
	PoolHash   []byte `gorm:"type:VARBINARY(28);not null;index"`
	TickerName string `gorm:"type:VARCHAR(5);not null"`
	Json       string `gorm:"type:TEXT;not null"`
	Bytes      []byte `gorm:"type:LONGBLOB;not null"`
	PmrHash    []byte `gorm:"type:VARBINARY(32);not null;index"`

	// Relationships
	Pool PoolHash        `gorm:"foreignKey:PoolHash;references:HashRaw"`
	Pmr  PoolMetadataRef `gorm:"foreignKey:PmrHash;references:Hash"`
}

func (OffChainPoolData) TableName() string {
	return "off_chain_pool_data"
}

// OffChainPoolFetchError represents pool metadata fetch errors with composite primary key
type OffChainPoolFetchError struct {
	PoolHash   []byte    `gorm:"type:VARBINARY(28);primaryKey"`
	PmrHash    []byte    `gorm:"type:VARBINARY(32);primaryKey"`
	FetchTime  time.Time `gorm:"primaryKey"`
	FetchError string    `gorm:"type:VARCHAR(255);not null"`
	RetryCount uint32    `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	Pool PoolHash        `gorm:"foreignKey:PoolHash;references:HashRaw"`
	PMR  PoolMetadataRef `gorm:"foreignKey:PmrHash;references:Hash"`
}

func (OffChainPoolFetchError) TableName() string {
	return "off_chain_pool_fetch_error"
}

// OffChainVoteData represents off-chain vote data with hash as primary key
type OffChainVoteData struct {
	Hash             []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	VotingAnchorHash []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Json             string  `gorm:"type:TEXT;not null"`
	Bytes            []byte  `gorm:"type:LONGBLOB;not null"`
	Warning          *string `gorm:"type:VARCHAR(255)"`
	Language         string  `gorm:"type:VARCHAR(255);not null"`
	Comment          *string `gorm:"type:TEXT"`
	IsValid          bool    `gorm:"not null"`

	// Relationships
	VotingAnchor    VotingAnchor                 `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
	GovActionData   []OffChainVoteGovActionData  `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
	DRepData        []OffChainVoteDRepData       `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
	Authors         []OffChainVoteAuthor         `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
	References      []OffChainVoteReference      `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
	ExternalUpdates []OffChainVoteExternalUpdate `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
}

func (OffChainVoteData) TableName() string {
	return "off_chain_vote_data"
}

// OffChainVoteGovActionData represents governance action data with composite primary key
type OffChainVoteGovActionData struct {
	OffChainVoteDataHash []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	Language             string  `gorm:"type:VARCHAR(255);primaryKey"`
	VotingAnchorHash     []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Title                *string `gorm:"type:TEXT"`
	Abstract             *string `gorm:"type:TEXT"`
	Motivation           *string `gorm:"type:TEXT"`
	Rationale            *string `gorm:"type:TEXT"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
	VotingAnchor     VotingAnchor     `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
}

func (OffChainVoteGovActionData) TableName() string {
	return "off_chain_vote_gov_action_data"
}

func (OffChainVoteDRepData) TableName() string {
	return "off_chain_vote_d_rep_data"
}

// OffChainVoteDRepData represents DRep off-chain data with composite primary key
type OffChainVoteDRepData struct {
	OffChainVoteDataHash []byte  `gorm:"type:VARBINARY(32);primaryKey"`
	DRepHashRaw          []byte  `gorm:"type:VARBINARY(28);primaryKey"`
	Language             string  `gorm:"type:VARCHAR(255);primaryKey"`
	VotingAnchorHash     []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Comment              *string `gorm:"type:TEXT"`
	Bio                  *string `gorm:"type:TEXT"`
	Email                *string `gorm:"type:VARCHAR(255)"`
	PaymentAddress       *string `gorm:"type:VARCHAR(255)"`
	GivenName            *string `gorm:"type:VARCHAR(255)"`
	Image                *string `gorm:"type:VARCHAR(255)"`
	Objectives           *string `gorm:"type:TEXT"`
	Motivations          *string `gorm:"type:TEXT"`
	Qualifications       *string `gorm:"type:TEXT"`
	DoNotList            bool    `gorm:"not null;default:false"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
	DRepHash         DRepHash         `gorm:"foreignKey:DRepHashRaw;references:HashRaw"`
	VotingAnchor     VotingAnchor     `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
}

// OffChainVoteAuthor represents vote authors with composite primary key
type OffChainVoteAuthor struct {
	Hash                 []byte  `gorm:"type:VARBINARY(32);primaryKey"` // Hash of vote data + name + witness
	OffChainVoteDataHash []byte  `gorm:"type:VARBINARY(32);not null;index"`
	Name                 *string `gorm:"type:VARCHAR(255)"`
	Witness              []byte  `gorm:"type:VARBINARY(32)"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
}

func (OffChainVoteExternalUpdate) TableName() string {
	return "off_chain_vote_external_updates"
}

func (OffChainVoteReference) TableName() string {
	return "off_chain_vote_references"
}

func (OffChainVoteAuthor) TableName() string {
	return "off_chain_vote_authors"
}

// GenerateOffChainVoteAuthorHash generates a unique hash for an author
func GenerateOffChainVoteAuthorHash(voteDataHash []byte, name *string, witness []byte) []byte {
	h := sha256.New()
	h.Write(voteDataHash)
	if name != nil {
		h.Write([]byte(*name))
	}
	if witness != nil {
		h.Write(witness)
	}
	return h.Sum(nil)
}

// OffChainVoteReference represents vote references with composite primary key
type OffChainVoteReference struct {
	OffChainVoteDataHash []byte `gorm:"type:VARBINARY(32);primaryKey"`
	Label                string `gorm:"type:VARCHAR(255);primaryKey"`
	URI                  string `gorm:"type:VARCHAR(255);not null"`
	ReferenceHash        []byte `gorm:"type:VARBINARY(32)"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
}

// OffChainVoteExternalUpdate represents external updates with composite primary key
type OffChainVoteExternalUpdate struct {
	OffChainVoteDataHash []byte `gorm:"type:VARBINARY(32);primaryKey"`
	Title                string `gorm:"type:VARCHAR(255);primaryKey"`
	URI                  string `gorm:"type:VARCHAR(255);not null"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataHash;references:Hash"`
}

// OffChainVoteFetchError represents vote fetch errors with composite primary key
type OffChainVoteFetchError struct {
	VotingAnchorHash []byte    `gorm:"type:VARBINARY(32);primaryKey"`
	FetchTime        time.Time `gorm:"primaryKey"`
	FetchError       string    `gorm:"type:VARCHAR(255);not null"`
	RetryCount       uint32    `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	VotingAnchor VotingAnchor `gorm:"foreignKey:VotingAnchorHash;references:Hash"`
}

func (OffChainVoteFetchError) TableName() string {
	return "off_chain_vote_fetch_errors"
}

// Remove all BeforeCreate hooks since we don't need ID management
func (o *OffChainPoolData) BeforeCreate(tx *gorm.DB) error           { return nil }
func (o *OffChainPoolFetchError) BeforeCreate(tx *gorm.DB) error     { return nil }
func (o *OffChainVoteData) BeforeCreate(tx *gorm.DB) error           { return nil }
func (o *OffChainVoteGovActionData) BeforeCreate(tx *gorm.DB) error  { return nil }
func (o *OffChainVoteDRepData) BeforeCreate(tx *gorm.DB) error       { return nil }
func (o *OffChainVoteAuthor) BeforeCreate(tx *gorm.DB) error         { return nil }
func (o *OffChainVoteReference) BeforeCreate(tx *gorm.DB) error      { return nil }
func (o *OffChainVoteExternalUpdate) BeforeCreate(tx *gorm.DB) error { return nil }
func (o *OffChainVoteFetchError) BeforeCreate(tx *gorm.DB) error     { return nil }
