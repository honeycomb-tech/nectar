package processors

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"nectar/database"
)

// MockTransactionInput implements ledger.TransactionInput
type MockTransactionInput struct {
	mock.Mock
	txHash string
	index  uint32
}

func (m *MockTransactionInput) Id() ledger.TransactionId {
	return ledger.ShelleyTransactionId{Hash: ledger.Blake2b256{}}
}

func (m *MockTransactionInput) Index() uint32 {
	return m.index
}

func (m *MockTransactionInput) Utxorpc() *any {
	return nil
}

// MockTransactionOutput implements ledger.TransactionOutput
type MockTransactionOutput struct {
	mock.Mock
	address ledger.Address
	amount  uint64
	assets  *ledger.MultiAsset[ledger.MultiAssetTypeOutput]
}

func (m *MockTransactionOutput) Address() ledger.Address {
	return m.address
}

func (m *MockTransactionOutput) Amount() uint64 {
	return m.amount
}

func (m *MockTransactionOutput) Assets() *ledger.MultiAsset[ledger.MultiAssetTypeOutput] {
	return m.assets
}

func (m *MockTransactionOutput) DatumHash() *ledger.Blake2b256 {
	return nil
}

func (m *MockTransactionOutput) Datum() *cbor.LazyValue {
	return nil
}

func (m *MockTransactionOutput) Script() ledger.ScriptRef {
	return ledger.ScriptRef{}
}

func (m *MockTransactionOutput) Utxorpc() *any {
	return nil
}

func setupTestDBForTx(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&database.Block{},
		&database.Transaction{},
		&database.TransactionInput{},
		&database.TransactionOutput{},
		&database.TransactionMetadata{},
		&database.MultiAsset{},
	)
	assert.NoError(t, err)

	return db
}

func TestNewTransactionProcessor(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.db)
}

func TestProcessTransaction_Basic(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Create a test block first
	block := &database.Block{
		Slot:    1000,
		BlockNo: 100,
		Hash:    "test_block_hash",
	}
	err := db.Create(block).Error
	assert.NoError(t, err)

	// Create mock transaction
	mockTx := &MockTransaction{
		hash:    "test_tx_hash",
		inputs:  []ledger.TransactionInput{},
		outputs: []ledger.TransactionOutput{},
	}

	tx, err := processor.ProcessTransaction(mockTx, block.ID, 0, 1)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "test_tx_hash", tx.Hash)
	assert.Equal(t, block.ID, tx.BlockID)
	assert.Equal(t, uint32(0), tx.BlockIndex)
}

func TestProcessTransaction_WithInputsOutputs(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Create a test block
	block := &database.Block{
		Slot:    2000,
		BlockNo: 200,
		Hash:    "block_with_io",
	}
	err := db.Create(block).Error
	assert.NoError(t, err)

	// Create mock inputs and outputs
	mockInput := &MockTransactionInput{
		txHash: "prev_tx_hash",
		index:  0,
	}

	mockOutput := &MockTransactionOutput{
		address: ledger.Address{},
		amount:  1000000,
	}

	mockTx := &MockTransaction{
		hash:    "tx_with_io",
		inputs:  []ledger.TransactionInput{mockInput},
		outputs: []ledger.TransactionOutput{mockOutput},
	}

	tx, err := processor.ProcessTransaction(mockTx, block.ID, 0, 1)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	// Verify inputs were saved
	var inputCount int64
	db.Model(&database.TransactionInput{}).Where("tx_id = ?", tx.ID).Count(&inputCount)
	assert.Equal(t, int64(1), inputCount)

	// Verify outputs were saved
	var outputCount int64
	db.Model(&database.TransactionOutput{}).Where("tx_id = ?", tx.ID).Count(&outputCount)
	assert.Equal(t, int64(1), outputCount)
}

func TestProcessTransaction_WithMetadata(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Create a test block
	block := &database.Block{
		Slot:    3000,
		BlockNo: 300,
		Hash:    "block_with_metadata",
	}
	err := db.Create(block).Error
	assert.NoError(t, err)

	// Create mock transaction with metadata
	mockTx := &MockTransaction{
		hash:     "tx_with_metadata",
		inputs:   []ledger.TransactionInput{},
		outputs:  []ledger.TransactionOutput{},
		metadata: map[uint64]interface{}{721: "NFT metadata"},
	}

	tx, err := processor.ProcessTransaction(mockTx, block.ID, 0, 1)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	// Verify metadata was saved
	var metadataCount int64
	db.Model(&database.TransactionMetadata{}).Where("tx_id = ?", tx.ID).Count(&metadataCount)
	assert.Equal(t, int64(1), metadataCount)
}

func TestProcessTransaction_DuplicateHandling(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Create a test block
	block := &database.Block{
		Slot:    4000,
		BlockNo: 400,
		Hash:    "block_duplicate",
	}
	err := db.Create(block).Error
	assert.NoError(t, err)

	mockTx := &MockTransaction{
		hash:    "duplicate_tx",
		inputs:  []ledger.TransactionInput{},
		outputs: []ledger.TransactionOutput{},
	}

	// Process transaction first time
	tx1, err := processor.ProcessTransaction(mockTx, block.ID, 0, 1)
	assert.NoError(t, err)
	assert.NotNil(t, tx1)

	// Process same transaction again
	tx2, err := processor.ProcessTransaction(mockTx, block.ID, 0, 1)
	assert.NoError(t, err)
	assert.NotNil(t, tx2)
	assert.Equal(t, tx1.ID, tx2.ID)

	// Verify only one transaction exists
	var count int64
	db.Model(&database.Transaction{}).Where("hash = ?", "duplicate_tx").Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestProcessTransactionInputs(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Create multiple mock inputs
	inputs := []ledger.TransactionInput{
		&MockTransactionInput{txHash: "tx1", index: 0},
		&MockTransactionInput{txHash: "tx2", index: 1},
		&MockTransactionInput{txHash: "tx3", index: 2},
	}

	err := processor.processTransactionInputs(db, inputs, 1, "test_tx_hash")
	assert.NoError(t, err)

	// Verify all inputs were saved
	var count int64
	db.Model(&database.TransactionInput{}).Where("tx_id = ?", 1).Count(&count)
	assert.Equal(t, int64(3), count)
}

func TestProcessTransactionOutputs(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Create multiple mock outputs with different amounts
	outputs := []ledger.TransactionOutput{
		&MockTransactionOutput{amount: 1000000},
		&MockTransactionOutput{amount: 2000000},
		&MockTransactionOutput{amount: 3000000},
	}

	err := processor.processTransactionOutputs(db, outputs, 1, "test_tx_hash")
	assert.NoError(t, err)

	// Verify all outputs were saved
	var savedOutputs []database.TransactionOutput
	db.Where("tx_id = ?", 1).Find(&savedOutputs)
	assert.Len(t, savedOutputs, 3)

	// Verify amounts
	assert.Equal(t, uint64(1000000), savedOutputs[0].Value)
	assert.Equal(t, uint64(2000000), savedOutputs[1].Value)
	assert.Equal(t, uint64(3000000), savedOutputs[2].Value)
}

func TestProcessTransaction_ErrorHandling(t *testing.T) {
	db := setupTestDBForTx(t)
	processor := NewTransactionProcessor(db)

	// Close database to force error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	mockTx := &MockTransaction{
		hash:    "error_tx",
		inputs:  []ledger.TransactionInput{},
		outputs: []ledger.TransactionOutput{},
	}

	_, err := processor.ProcessTransaction(mockTx, 1, 0, 1)
	assert.Error(t, err)
}