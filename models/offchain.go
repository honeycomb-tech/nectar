package models

import (
	"time"
)

// OffChainPoolData represents off-chain pool metadata
type OffChainPoolData struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	PoolID      uint64    `gorm:"not null"`
	TickerName  string    `gorm:"type:VARCHAR(5);not null"`
	Hash        []byte    `gorm:"type:VARBINARY(32);not null"`
	Json        string    `gorm:"type:TEXT;not null"`
	Bytes       []byte    `gorm:"type:LONGBLOB;not null"`
	PmrID       uint64    `gorm:"not null"`

	// Relationships
	Pool PoolHash `gorm:"foreignKey:PoolID"`
	Pmr  PoolMetadataRef `gorm:"foreignKey:PmrID"`
}

// OffChainPoolFetchError represents pool metadata fetch errors
type OffChainPoolFetchError struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	PoolID      uint64    `gorm:"not null"`
	FetchError  string    `gorm:"type:VARCHAR(255);not null"`
	FetchTime   time.Time `gorm:"not null"`
	RetryCount  uint32    `gorm:"type:INT UNSIGNED;not null"`
	PmrID       uint64    `gorm:"not null"`

	// Relationships
	Pool PoolMetadataRef `gorm:"foreignKey:PoolID"`
	PMR  PoolMetadataRef `gorm:"foreignKey:PmrID"`
}

// OffChainVoteData represents off-chain vote data
type OffChainVoteData struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	VotingAnchorID  uint64 `gorm:"not null"`
	Hash            []byte `gorm:"type:VARBINARY(32);not null"`
	Json            string `gorm:"type:TEXT;not null"`
	Bytes           []byte `gorm:"type:LONGBLOB;not null"`
	Warning         *string `gorm:"type:VARCHAR(255)"`
	Language        string `gorm:"type:VARCHAR(255);not null"`
	Comment         *string `gorm:"type:TEXT"`
	IsValid         bool   `gorm:"not null"`

	// Relationships
	VotingAnchor         VotingAnchor         `gorm:"foreignKey:VotingAnchorID"`
	GovActionData        []OffChainVoteGovActionData `gorm:"foreignKey:OffChainVoteDataID"`
	DRepData             []OffChainVoteDRepData      `gorm:"foreignKey:OffChainVoteDataID"`
	Authors              []OffChainVoteAuthor        `gorm:"foreignKey:OffChainVoteDataID"`
	References           []OffChainVoteReference     `gorm:"foreignKey:OffChainVoteDataID"`
	ExternalUpdates      []OffChainVoteExternalUpdate `gorm:"foreignKey:OffChainVoteDataID"`
}

// OffChainVoteGovActionData represents governance action data
type OffChainVoteGovActionData struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement"`
	OffChainVoteDataID  uint64 `gorm:"not null"`
	VotingAnchorID      uint64 `gorm:"not null"`
	Language            string `gorm:"type:VARCHAR(255);not null"`
	Title               *string `gorm:"type:TEXT"`
	Abstract            *string `gorm:"type:TEXT"`
	Motivation          *string `gorm:"type:TEXT"`
	Rationale           *string `gorm:"type:TEXT"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataID"`
	VotingAnchor     VotingAnchor     `gorm:"foreignKey:VotingAnchorID"`
}

// OffChainVoteDRepData represents DRep off-chain data
type OffChainVoteDRepData struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement"`
	OffChainVoteDataID  uint64 `gorm:"not null"`
	DRepHashID          uint64 `gorm:"not null"`
	VotingAnchorID      uint64 `gorm:"not null"`
	Language            string `gorm:"type:VARCHAR(255);not null"`
	Comment             *string `gorm:"type:TEXT"`
	Bio                 *string `gorm:"type:TEXT"`
	Email               *string `gorm:"type:VARCHAR(255)"`
	PaymentAddress      *string `gorm:"type:VARCHAR(255)"`
	GivenName           *string `gorm:"type:VARCHAR(255)"`
	Image               *string `gorm:"type:VARCHAR(255)"`
	Objectives          *string `gorm:"type:TEXT"`
	Motivations         *string `gorm:"type:TEXT"`
	Qualifications      *string `gorm:"type:TEXT"`
	DoNotList           bool    `gorm:"not null;default:false"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataID"`
	DRepHash         DRepHash         `gorm:"foreignKey:DRepHashID"`
	VotingAnchor     VotingAnchor     `gorm:"foreignKey:VotingAnchorID"`
}

// OffChainVoteAuthor represents vote authors
type OffChainVoteAuthor struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement"`
	OffChainVoteDataID  uint64 `gorm:"not null"`
	Name                *string `gorm:"type:VARCHAR(255)"`
	Witness             []byte  `gorm:"type:VARBINARY(32)"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataID"`
}

// OffChainVoteReference represents vote references
type OffChainVoteReference struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement"`
	OffChainVoteDataID  uint64 `gorm:"not null"`
	Label               string `gorm:"type:VARCHAR(255);not null"`
	URI                 string `gorm:"type:VARCHAR(255);not null"`
	ReferenceHash       []byte `gorm:"type:VARBINARY(32)"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataID"`
}

// OffChainVoteExternalUpdate represents external updates
type OffChainVoteExternalUpdate struct {
	ID                  uint64 `gorm:"primaryKey;autoIncrement"`
	OffChainVoteDataID  uint64 `gorm:"not null"`
	Title               string `gorm:"type:VARCHAR(255);not null"`
	URI                 string `gorm:"type:VARCHAR(255);not null"`

	// Relationships
	OffChainVoteData OffChainVoteData `gorm:"foreignKey:OffChainVoteDataID"`
}

// OffChainVoteFetchError represents vote fetch errors
type OffChainVoteFetchError struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	VotingAnchorID uint64    `gorm:"not null"`
	FetchError     string    `gorm:"type:VARCHAR(255);not null"`
	FetchTime      time.Time `gorm:"not null"`
	RetryCount     uint32    `gorm:"type:INT UNSIGNED;not null"`

	// Relationships
	VotingAnchor VotingAnchor `gorm:"foreignKey:VotingAnchorID"`
} 