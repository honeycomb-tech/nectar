package processors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"nectar/database"
)

// MockBlock implements ledger.Block interface for testing
type MockBlock struct {
	mock.Mock
	slot       uint64
	number     uint64
	hash       string
	era        uint
	txs        []ledger.Transaction
	blockBody  interface{}
}

func (m *MockBlock) SlotNumber() uint64 {
	return m.slot
}

func (m *MockBlock) BlockNumber() uint64 {
	return m.number
}

func (m *MockBlock) Hash() string {
	return m.hash
}

func (m *MockBlock) Era() ledger.Era {
	return ledger.Era{Id: m.era}
}

func (m *MockBlock) Transactions() []ledger.Transaction {
	return m.txs
}

func (m *MockBlock) Utxorpc() *any {
	return nil
}

func (m *MockBlock) BlockBodyCbor() []byte {
	return []byte{}
}

func (m *MockBlock) IssuerVkey() ledger.IssuerVkey {
	return ledger.IssuerVkey{}
}

func (m *MockBlock) ProtocolVersion() ledger.ProtocolVersion {
	return ledger.ProtocolVersion{}
}

func (m *MockBlock) VrfKey() []byte {
	return []byte{}
}

func (m *MockBlock) BlockBody() interface{} {
	return m.blockBody
}

// MockTransaction implements ledger.Transaction interface
type MockTransaction struct {
	mock.Mock
	hash     string
	inputs   []ledger.TransactionInput
	outputs  []ledger.TransactionOutput
	metadata interface{}
}

func (m *MockTransaction) Hash() string {
	return m.hash
}

func (m *MockTransaction) Inputs() []ledger.TransactionInput {
	return m.inputs
}

func (m *MockTransaction) Outputs() []ledger.TransactionOutput {
	return m.outputs
}

func (m *MockTransaction) Fee() uint64 {
	return 1000000
}

func (m *MockTransaction) TTL() uint64 {
	return 0
}

func (m *MockTransaction) ValidityIntervalStart() uint64 {
	return 0
}

func (m *MockTransaction) ProtocolParametersUpdate() map[ledger.Blake2b224]any {
	return nil
}

func (m *MockTransaction) ReferenceInputs() []ledger.TransactionInput {
	return nil
}

func (m *MockTransaction) Collateral() []ledger.TransactionInput {
	return nil
}

func (m *MockTransaction) CollateralReturn() *ledger.TransactionOutput {
	return nil
}

func (m *MockTransaction) TotalCollateral() uint64 {
	return 0
}

func (m *MockTransaction) Certificates() []ledger.Certificate {
	return nil
}

func (m *MockTransaction) Withdrawals() map[*ledger.Address]uint64 {
	return nil
}

func (m *MockTransaction) Mint() *ledger.MultiAsset[ledger.MultiAssetTypeMint] {
	return nil
}

func (m *MockTransaction) ScriptDataHash() *ledger.Blake2b256 {
	return nil
}

func (m *MockTransaction) VotingProcedures() ledger.VotingProcedures {
	return ledger.VotingProcedures{}
}

func (m *MockTransaction) ProposalProcedures() []ledger.ProposalProcedure {
	return nil
}

func (m *MockTransaction) CurrentTreasuryValue() int64 {
	return 0
}

func (m *MockTransaction) Donation() uint64 {
	return 0
}

func (m *MockTransaction) Metadata() *cbor.LazyValue {
	if m.metadata != nil {
		return &cbor.LazyValue{}
	}
	return nil
}

func (m *MockTransaction) IsValid() bool {
	return true
}

func (m *MockTransaction) Consumed() []ledger.TransactionInput {
	return m.inputs
}

func (m *MockTransaction) Produced() []ledger.TransactionOutput {
	return m.outputs
}

func (m *MockTransaction) UtxorpcTx() *any {
	return nil
}

// Test helper functions
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&database.Block{},
		&database.Transaction{},
		&database.TransactionInput{},
		&database.TransactionOutput{},
		&database.StakeRegistration{},
		&database.StakeDeregistration{},
		&database.PoolRegistration{},
		&database.PoolRetirement{},
		&database.Delegation{},
		&database.Withdrawal{},
		&database.TransactionMetadata{},
		&database.Script{},
		&database.Redeemer{},
		&database.Datum{},
		&database.CollateralInput{},
		&database.CollateralOutput{},
		&database.ReferenceInput{},
		&database.GovernanceAction{},
		&database.GovernanceVote{},
		&database.DRepRegistration{},
		&database.DRepDeregistration{},
		&database.DRepUpdate{},
		&database.CommitteeHotKeyRegistration{},
		&database.CommitteeHotKeyDeregistration{},
		&database.CommitteeResignation{},
	)
	assert.NoError(t, err)

	return db
}

func TestNewBlockProcessor(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.db)
	assert.NotNil(t, processor.txProcessor)
	assert.NotNil(t, processor.certProcessor)
	assert.NotNil(t, processor.withdrawalProcessor)
	assert.NotNil(t, processor.metadataProcessor)
	assert.NotNil(t, processor.scriptProcessor)
	assert.NotNil(t, processor.govProcessor)
	assert.NotNil(t, processor.stakeCache)
}

func TestProcessBlock_Byron(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	// Create a mock Byron block
	block := &MockBlock{
		slot:   1000,
		number: 100,
		hash:   "byron_block_hash",
		era:    0, // Byron era
		txs:    []ledger.Transaction{},
		blockBody: &byron.ByronMainBlockBody{
			TxPayload: []byron.ByronTx{},
		},
	}

	ctx := context.Background()
	err := processor.ProcessBlock(ctx, block, 0)
	assert.NoError(t, err)

	// Verify block was saved
	var savedBlock database.Block
	err = db.First(&savedBlock, "hash = ?", "byron_block_hash").Error
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), savedBlock.Slot)
	assert.Equal(t, uint64(100), savedBlock.BlockNo)
	assert.Equal(t, "byron_block_hash", savedBlock.Hash)
}

func TestProcessBlock_Shelley(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	// Create mock Shelley block with transaction
	mockTx := &MockTransaction{
		hash:    "tx_hash_1",
		inputs:  []ledger.TransactionInput{},
		outputs: []ledger.TransactionOutput{},
	}

	block := &MockBlock{
		slot:   5000000,
		number: 200000,
		hash:   "shelley_block_hash",
		era:    1, // Shelley era
		txs:    []ledger.Transaction{mockTx},
		blockBody: &shelley.ShelleyBlockBody{
			TransactionBodies: []shelley.ShelleyTransactionBody{},
		},
	}

	ctx := context.Background()
	err := processor.ProcessBlock(ctx, block, 1)
	assert.NoError(t, err)

	// Verify block was saved
	var savedBlock database.Block
	err = db.First(&savedBlock, "hash = ?", "shelley_block_hash").Error
	assert.NoError(t, err)
	assert.Equal(t, uint64(5000000), savedBlock.Slot)
	assert.Equal(t, uint64(200000), savedBlock.BlockNo)
}

func TestProcessBlock_DuplicateBlock(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	block := &MockBlock{
		slot:   1000,
		number: 100,
		hash:   "duplicate_block_hash",
		era:    0,
		txs:    []ledger.Transaction{},
	}

	ctx := context.Background()
	
	// Process block first time
	err := processor.ProcessBlock(ctx, block, 0)
	assert.NoError(t, err)

	// Process same block again - should handle duplicate gracefully
	err = processor.ProcessBlock(ctx, block, 0)
	assert.NoError(t, err)

	// Verify only one block exists
	var count int64
	db.Model(&database.Block{}).Where("hash = ?", "duplicate_block_hash").Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestProcessBlock_WithTransactions(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	// Create mock transactions
	tx1 := &MockTransaction{
		hash:    "tx_hash_1",
		inputs:  []ledger.TransactionInput{},
		outputs: []ledger.TransactionOutput{},
	}
	tx2 := &MockTransaction{
		hash:    "tx_hash_2",
		inputs:  []ledger.TransactionInput{},
		outputs: []ledger.TransactionOutput{},
	}

	block := &MockBlock{
		slot:   2000,
		number: 200,
		hash:   "block_with_txs",
		era:    1,
		txs:    []ledger.Transaction{tx1, tx2},
	}

	ctx := context.Background()
	err := processor.ProcessBlock(ctx, block, 1)
	assert.NoError(t, err)

	// Verify transactions were saved
	var txCount int64
	db.Model(&database.Transaction{}).Where("block_id = ?", 200).Count(&txCount)
	assert.Equal(t, int64(2), txCount)
}

func TestProcessBlock_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	block := &MockBlock{
		slot:   3000,
		number: 300,
		hash:   "cancelled_block",
		era:    1,
		txs:    []ledger.Transaction{},
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := processor.ProcessBlock(ctx, block, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestEnsureHealthyConnection(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	err := processor.EnsureHealthyConnection()
	assert.NoError(t, err)
}

func TestGetEraConfig(t *testing.T) {
	tests := []struct {
		name     string
		epochNo  uint64
		expected string
	}{
		{"Byron Era", 100, "Byron"},
		{"Shelley Era", 210, "Shelley"},
		{"Allegra Era", 240, "Allegra"},
		{"Mary Era", 260, "Mary"},
		{"Alonzo Era", 300, "Alonzo"},
		{"Babbage Era", 370, "Babbage"},
		{"Conway Era", 450, "Conway"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetEraConfig(tt.epochNo)
			assert.Equal(t, tt.expected, config.Name)
			assert.True(t, config.MaxTxSize > 0)
			assert.True(t, config.MaxBlockSize > 0)
		})
	}
}

func TestProcessBlock_Performance(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	// Create a block with many transactions
	var txs []ledger.Transaction
	for i := 0; i < 100; i++ {
		tx := &MockTransaction{
			hash:    fmt.Sprintf("tx_hash_%d", i),
			inputs:  []ledger.TransactionInput{},
			outputs: []ledger.TransactionOutput{},
		}
		txs = append(txs, tx)
	}

	block := &MockBlock{
		slot:   4000,
		number: 400,
		hash:   "performance_test_block",
		era:    1,
		txs:    txs,
	}

	ctx := context.Background()
	start := time.Now()
	err := processor.ProcessBlock(ctx, block, 1)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, 5*time.Second, "Block processing took too long")

	// Verify all transactions were saved
	var txCount int64
	db.Model(&database.Transaction{}).Where("block_id = ?", 400).Count(&txCount)
	assert.Equal(t, int64(100), txCount)
}

func TestProcessBlock_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	processor := NewBlockProcessor(db)

	// Close the database to force an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	block := &MockBlock{
		slot:   5000,
		number: 500,
		hash:   "error_block",
		era:    1,
		txs:    []ledger.Transaction{},
	}

	ctx := context.Background()
	err := processor.ProcessBlock(ctx, block, 1)
	assert.Error(t, err)
}