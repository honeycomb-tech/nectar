package processors

import (
	"context"
	"fmt"
	"log"
	unifiederrors "nectar/errors"
	"nectar/models"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Block type constants
const (
	BlockTypeByronEBB  = 0
	BlockTypeByronMain = 1
)

// Optimal batch sizes for different database operations
const (
	TRANSACTION_BATCH_SIZE = 2000   // Increased for TiDB
	TX_OUT_BATCH_SIZE      = 10000  // TiDB handles large batches well
	TX_IN_BATCH_SIZE       = 5000   // Increased for performance
	METADATA_BATCH_SIZE    = 1000   // Larger metadata batches
	BLOCK_BATCH_SIZE       = 500    // More blocks per batch
)

// BlockProcessor handles processing blocks and storing them in TiDB
type BlockProcessor struct {
	db                   *gorm.DB
	stakeAddressCache    *StakeAddressCache
	errorCollector       *ErrorCollector
	certificateProcessor *CertificateProcessor
	withdrawalProcessor  *WithdrawalProcessor
	assetProcessor       *AssetProcessor
	metadataProcessor    *MetadataProcessor
	governanceProcessor  *GovernanceProcessor
	scriptProcessor      *ScriptProcessor
	metadataFetcher      MetadataFetcher
}

// NewBlockProcessor creates a new block processor
func NewBlockProcessor(db *gorm.DB) *BlockProcessor {
	stakeAddressCache := NewStakeAddressCache(db)
	bp := &BlockProcessor{
		db:                   db,
		stakeAddressCache:    stakeAddressCache,
		errorCollector:       GetGlobalErrorCollector(),
		certificateProcessor: NewCertificateProcessor(db, stakeAddressCache),
		withdrawalProcessor:  NewWithdrawalProcessor(db),
		assetProcessor:       NewAssetProcessor(db),
		metadataProcessor:    NewMetadataProcessor(db),
		governanceProcessor:  NewGovernanceProcessor(db),
		scriptProcessor:      NewScriptProcessor(db),
	}
	return bp
}

// ProcessBlock processes a single block and its transactions
func (bp *BlockProcessor) ProcessBlock(ctx context.Context, block ledger.Block, blockType uint) error {
	return bp.db.Transaction(func(tx *gorm.DB) error {
		// Process the block header
		dbBlock, err := bp.processBlockHeader(ctx, tx, block, blockType)
		if err != nil {
			return fmt.Errorf("failed to process block header: %w", err)
		}

		// Process transactions if any
		if err := bp.processEraAwareTransactions(ctx, tx, dbBlock, block, blockType); err != nil {
			return fmt.Errorf("failed to process transactions: %w", err)
		}

		return nil
	})
}

// processBlockHeader processes and stores the block header with hash-based primary key
func (bp *BlockProcessor) processBlockHeader(ctx context.Context, tx *gorm.DB, block ledger.Block, blockType uint) (*models.Block, error) {
	// Extract block data
	blockHeader := block.Header()
	hash := blockHeader.Hash()
	hashBytes := hash[:]
	
	slotNumber := blockHeader.SlotNumber()
	epochNo := bp.getEpochForSlot(slotNumber, blockType)
	blockNumber64 := block.BlockNumber()
	blockNumber := uint32(blockNumber64)

	// Get slot leader hash
	slotLeaderHash, err := bp.getOrCreateSlotLeader(tx, block, blockType)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot leader: %w", err)
	}

	// Create the block record with hash as primary key
	dbBlock := &models.Block{
		Hash:           hashBytes,
		SlotNo:         &slotNumber,
		EpochNo:        &epochNo,
		BlockNo:        &blockNumber,
		SlotLeaderHash: slotLeaderHash,
		Size:           uint32(block.BlockBodySize()),
		Time:           bp.getBlockTime(slotNumber, blockType),
		TxCount:        uint64(len(block.Transactions())),
		ProtoMajor:     0, // TODO: Extract from block header based on era
		ProtoMinor:     0, // TODO: Extract from block header based on era
		Era:            bp.getEraName(blockType),
	}

	// Set previous block hash if not genesis
	if blockNumber64 > 0 {
		prevHash := block.Header().PrevHash()
		// Convert Blake2b256 to []byte
		if !isZeroHash(prevHash) {
			dbBlock.PreviousHash = prevHash[:]
		}
	}

	// Set era-specific fields
	bp.setEraSpecificBlockFields(dbBlock, block, blockType)

	// Insert block with duplicate handling
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}},
		DoNothing: true,
	}).Create(dbBlock).Error; err != nil {
		// Check if it's a duplicate (already exists)
		var existingBlock models.Block
		if err2 := tx.Where("hash = ?", hashBytes).First(&existingBlock).Error; err2 == nil {
			// Block already exists, use it
			return &existingBlock, nil
		}
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	// Check for epoch boundary
	if bp.isEpochBoundary(slotNumber, blockType) {
		if err := bp.processEpochBoundary(ctx, tx, dbBlock, epochNo, blockType); err != nil {
			log.Printf("[WARNING] Failed to process epoch boundary at slot %d: %v", slotNumber, err)
		}
	}

	return dbBlock, nil
}

// processEraAwareTransactions processes transactions with era-specific handling
func (bp *BlockProcessor) processEraAwareTransactions(ctx context.Context, tx *gorm.DB, dbBlock *models.Block, block ledger.Block, blockType uint) error {
	transactions := block.Transactions()
	if len(transactions) == 0 {
		return nil
	}

	// Process transactions in batches
	// Increased from 50 to 500 for better TiDB performance
	batchSize := 500
	for i := 0; i < len(transactions); i += batchSize {
		end := i + batchSize
		if end > len(transactions) {
			end = len(transactions)
		}

		if err := bp.processTransactionBatch(ctx, tx, dbBlock.Hash, transactions[i:end], i, blockType); err != nil {
			return fmt.Errorf("failed to process transaction batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// processTransactionBatch processes a batch of transactions with hash-based keys
func (bp *BlockProcessor) processTransactionBatch(ctx context.Context, tx *gorm.DB, blockHash []byte, transactions []ledger.Transaction, startIndex int, blockType uint) error {
	txBatch := make([]models.Tx, 0, len(transactions))

	// First pass: create transaction records
	for i, transaction := range transactions {
		blockIndex := startIndex + i
		hash := transaction.Hash()
		hashBytes := hash[:]

		dbTx := models.Tx{
			Hash:       hashBytes,
			BlockHash:  blockHash,
			BlockIndex: uint32(blockIndex),
			Size:       uint32(len(transaction.Cbor())),
		}

		// Set basic transaction fields
		if fee := transaction.Fee(); fee > 0 {
			fee64 := int64(fee)
			dbTx.Fee = &fee64
		}

		// Calculate output sum
		if outputs := transaction.Outputs(); len(outputs) > 0 {
			var outSum int64
			for _, output := range outputs {
				outSum += int64(output.Amount())
			}
			dbTx.OutSum = &outSum
		}

		// Era-specific transaction processing
		bp.setEraSpecificTxFields(&dbTx, transaction, blockType)

		txBatch = append(txBatch, dbTx)
	}

	// Batch insert transactions
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}},
		DoNothing: true,
	}).CreateInBatches(txBatch, TRANSACTION_BATCH_SIZE).Error; err != nil {
		return fmt.Errorf("failed to batch insert transactions: %w", err)
	}

	// Second pass: process transaction components
	for i, transaction := range transactions {
		hash := transaction.Hash()
		txHash := hash[:]

		// Process transaction inputs
		if err := bp.processTransactionInputs(ctx, tx, txHash, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessInputs", fmt.Sprintf("Failed to process inputs for tx %d: %v", i, err))
		}

		// Process transaction outputs
		if err := bp.processTransactionOutputs(ctx, tx, txHash, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessOutputs", fmt.Sprintf("Failed to process outputs for tx %d: %v", i, err))
		}

		// Process metadata
		if metadata := transaction.Metadata(); metadata != nil {
			// Check if metadata has any content
			if metaValue := metadata.Value(); metaValue != nil {
				if err := bp.metadataProcessor.ProcessMetadata(tx, txHash, metadata); err != nil {
					unifiederrors.Get().Warning("BlockProcessor", "ProcessMetadata", fmt.Sprintf("Failed to process metadata: %v", err))
				}
			}
		}

		// Process certificates
		if certs := bp.getTransactionCertificates(transaction, blockType); len(certs) > 0 {
			if err := bp.certificateProcessor.ProcessCertificates(ctx, tx, txHash, certs); err != nil {
				unifiederrors.Get().Warning("BlockProcessor", "ProcessCertificates", fmt.Sprintf("Failed to process certificates: %v", err))
			}
		}

		// Process withdrawals
		if withdrawals := bp.getTransactionWithdrawals(transaction, blockType); len(withdrawals) > 0 {
			if err := bp.withdrawalProcessor.ProcessWithdrawals(ctx, tx, txHash, withdrawals, blockType); err != nil {
				unifiederrors.Get().Warning("BlockProcessor", "ProcessWithdrawals", fmt.Sprintf("Failed to process withdrawals: %v", err))
			}
		}

		// Process minting/burning
		if mint := bp.getTransactionMint(transaction, blockType); len(mint) > 0 {
			if err := bp.assetProcessor.ProcessMint(tx, txHash, mint); err != nil {
				unifiederrors.Get().Warning("BlockProcessor", "ProcessMint", fmt.Sprintf("Failed to process mint: %v", err))
			}
		}

		// Process governance actions
		if bp.hasGovernanceData(transaction, blockType) {
			if err := bp.governanceProcessor.ProcessTransaction(tx, txHash, transaction, blockType); err != nil {
				unifiederrors.Get().Warning("BlockProcessor", "ProcessGovernance", fmt.Sprintf("Failed to process governance: %v", err))
			}
		}

		// Process scripts
		if err := bp.scriptProcessor.ProcessTransaction(tx, txHash, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessScripts", fmt.Sprintf("Failed to process scripts: %v", err))
		}
	}

	return nil
}

// processTransactionInputs processes transaction inputs with hash references
func (bp *BlockProcessor) processTransactionInputs(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, blockType uint) error {
	inputs := transaction.Inputs()
	if len(inputs) == 0 {
		return nil
	}

	inputBatch := make([]models.TxIn, 0, len(inputs))

	for i, input := range inputs {
		// Get the referenced transaction output
		id := input.Id()
		outTxHash := id[:]
		outIndex := input.Index()

		dbInput := models.TxIn{
			TxInHash:   txHash,
			TxInIndex:  uint32(i),
			TxOutHash:  outTxHash,
			TxOutIndex: outIndex,
		}

		// Handle redeemer if present
		if redeemer := bp.getInputRedeemer(transaction, i, blockType); redeemer != nil {
			redeemerHash := models.GenerateRedeemerHash(txHash, "spend", uint32(i))
			dbInput.RedeemerHash = redeemerHash
		}

		inputBatch = append(inputBatch, dbInput)
	}

	// Batch insert inputs
	if err := tx.CreateInBatches(inputBatch, TX_IN_BATCH_SIZE).Error; err != nil {
		return fmt.Errorf("failed to batch insert inputs: %w", err)
	}

	return nil
}

// processTransactionOutputs processes transaction outputs with composite keys
func (bp *BlockProcessor) processTransactionOutputs(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, blockType uint) error {
	outputs := transaction.Outputs()
	if len(outputs) == 0 {
		return nil
	}

	outputBatch := make([]models.TxOut, 0, len(outputs))

	for i, output := range outputs {
		address := output.Address()
		addressStr := address.String()
		addressBytes, err := address.Bytes()
		if err != nil {
			// Use empty bytes on error
			addressBytes = []byte{}
		}

		dbOutput := models.TxOut{
			TxHash:     txHash,
			Index:      uint32(i),
			Address:    addressStr,
			AddressRaw: addressBytes,
			Value:      output.Amount(),
		}

		// Extract payment credential
		if paymentCred := bp.extractPaymentCredential(address); paymentCred != nil {
			dbOutput.PaymentCred = paymentCred
		}

		// Extract stake address
		// TODO: Implement stake address extraction
		// For now, skip stake address processing

		// Handle datum hash
		if datumHash := output.DatumHash(); datumHash != nil {
			dbOutput.DataHash = datumHash.Bytes()
		}

		// Handle inline datum (Babbage+)
		if inlineDatum := bp.getOutputInlineDatum(output, blockType); inlineDatum != nil {
			dbOutput.InlineDatumHash = inlineDatum
		}

		// Handle reference script (Babbage+)
		if refScript := bp.getOutputReferenceScript(output, blockType); refScript != nil {
			dbOutput.ReferenceScriptHash = refScript
		}

		outputBatch = append(outputBatch, dbOutput)
	}

	// Batch insert outputs
	// Use larger batch size if we have a lot of outputs
	batchSize := TX_OUT_BATCH_SIZE
	if len(outputBatch) > TX_OUT_BATCH_SIZE*2 {
		batchSize = TX_OUT_BATCH_SIZE * 2 // Double batch size for large sets
	}
	
	if err := tx.CreateInBatches(outputBatch, batchSize).Error; err != nil {
		return fmt.Errorf("failed to batch insert outputs: %w", err)
	}

	// Process multi-assets in outputs
	for i, output := range outputs {
		if assets := output.Assets(); assets != nil {
			if err := bp.assetProcessor.ProcessOutputAssets(tx, txHash, uint32(i), assets); err != nil {
				return fmt.Errorf("failed to process output %d assets: %w", i, err)
			}
		}
	}

	return nil
}

// getOrCreateSlotLeader gets or creates a slot leader by hash
func (bp *BlockProcessor) getOrCreateSlotLeader(tx *gorm.DB, block ledger.Block, blockType uint) ([]byte, error) {
	// For Byron blocks, use a deterministic hash based on block data
	if blockType == BlockTypeByronEBB || blockType == BlockTypeByronMain {
		// Create a deterministic hash for Byron slot leader
		hash := block.Header().Hash()
		slotLeaderHash := hash[:28] // Use first 28 bytes of block hash

		slotLeader := &models.SlotLeader{
			Hash:        slotLeaderHash,
			Description: stringPtr("Byron slot leader"),
		}

		// Insert or ignore if exists
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hash"}},
			DoNothing: true,
		}).Create(slotLeader).Error; err != nil {
			return nil, err
		}

		return slotLeaderHash, nil
	}

	// For Shelley+ blocks, extract issuer verification key
	blockHeader := block.Header()
	var issuerVkey []byte

	switch header := blockHeader.(type) {
	case *ledger.ShelleyBlockHeader:
		issuerVkey = header.Body.IssuerVkey[:]
	case *ledger.BabbageBlockHeader:
		issuerVkey = header.Body.IssuerVkey[:]
	default:
		// Fallback to block hash
		hash := blockHeader.Hash()
		issuerVkey = hash[:28]
	}

	// Use first 28 bytes as slot leader hash
	slotLeaderHash := issuerVkey[:28]

	slotLeader := &models.SlotLeader{
		Hash: slotLeaderHash,
	}

	// TODO: Look up pool hash from issuer vkey if needed
	// This would require additional stake pool registration tracking

	// Insert or ignore if exists
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}},
		DoNothing: true,
	}).Create(slotLeader).Error; err != nil {
		return nil, err
	}

	return slotLeaderHash, nil
}

// Helper functions remain largely the same but work with hashes instead of IDs
func (bp *BlockProcessor) getEpochForSlot(slot uint64, blockType uint) uint32 {
	switch blockType {
	case BlockTypeByronEBB, BlockTypeByronMain:
		return uint32(slot / 21600) // Byron epoch = 21600 slots
	default:
		return uint32(slot / 432000) // Shelley+ epoch = 432000 slots
	}
}

func (bp *BlockProcessor) getBlockTime(slot uint64, blockType uint) time.Time {
	// Shelley start time: July 29, 2020 21:44:51 UTC
	shelleyStart := time.Date(2020, 7, 29, 21, 44, 51, 0, time.UTC)
	
	switch blockType {
	case BlockTypeByronEBB, BlockTypeByronMain:
		// Byron era: 20 second slots, started at mainnet launch
		byronStart := time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)
		return byronStart.Add(time.Duration(slot) * 20 * time.Second)
	default:
		// Shelley+ era: 1 second slots
		// Adjust for Byron slots (4492800 Byron slots at 20s each)
		shelleySlot := slot - 4492800
		return shelleyStart.Add(time.Duration(shelleySlot) * time.Second)
	}
}

func (bp *BlockProcessor) getEraName(blockType uint) string {
	switch blockType {
	case BlockTypeByronEBB, BlockTypeByronMain:
		return "Byron"
	case 2, 3: // Shelley
		return "Shelley"
	case 4: // Allegra
		return "Allegra"
	case 5: // Mary
		return "Mary"
	case 6: // Alonzo
		return "Alonzo"
	case 7: // Babbage
		return "Babbage"
	case 8: // Conway
		return "Conway"
	default:
		return "Unknown"
	}
}

func (bp *BlockProcessor) isEpochBoundary(slot uint64, blockType uint) bool {
	switch blockType {
	case BlockTypeByronEBB:
		return true // EBB blocks are by definition epoch boundaries
	case BlockTypeByronMain:
		return slot%21600 == 0
	default:
		return slot%432000 == 0
	}
}

// Stub functions for era-specific processing
func (bp *BlockProcessor) setEraSpecificBlockFields(block *models.Block, ledgerBlock ledger.Block, blockType uint) {
	// TODO: Implement era-specific block fields
}

func (bp *BlockProcessor) setEraSpecificTxFields(tx *models.Tx, ledgerTx ledger.Transaction, blockType uint) {
	// TODO: Implement era-specific transaction fields
}

func (bp *BlockProcessor) processEpochBoundary(ctx context.Context, tx *gorm.DB, block *models.Block, epochNo uint32, blockType uint) error {
	// TODO: Implement epoch boundary processing
	return nil
}

func (bp *BlockProcessor) getTransactionCertificates(tx ledger.Transaction, blockType uint) []interface{} {
	// TODO: Extract certificates based on era
	return nil
}

func (bp *BlockProcessor) getTransactionWithdrawals(tx ledger.Transaction, blockType uint) map[string]uint64 {
	// TODO: Extract withdrawals based on era
	// Note: Will need to convert ledger.Address to string when implementing
	return nil
}

func (bp *BlockProcessor) getTransactionMint(tx ledger.Transaction, blockType uint) map[string]uint64 {
	// TODO: Extract mint based on era
	return nil
}

func (bp *BlockProcessor) hasGovernanceData(tx ledger.Transaction, blockType uint) bool {
	// TODO: Check for governance data based on era
	return false
}

func (bp *BlockProcessor) getInputRedeemer(tx ledger.Transaction, index int, blockType uint) interface{} {
	// TODO: Extract redeemer based on era
	return nil
}

func (bp *BlockProcessor) getOutputInlineDatum(output ledger.TransactionOutput, blockType uint) []byte {
	// TODO: Extract inline datum for Babbage+ eras
	return nil
}

func (bp *BlockProcessor) getOutputReferenceScript(output ledger.TransactionOutput, blockType uint) []byte {
	// TODO: Extract reference script for Babbage+ eras
	return nil
}

func (bp *BlockProcessor) extractPaymentCredential(addr ledger.Address) []byte {
	// TODO: Extract payment credential from address
	return nil
}

func (bp *BlockProcessor) extractStakeAddress(addr ledger.Address) ledger.Address {
	// TODO: Extract stake address from address
	// Return a typed nil address
	var nilAddr ledger.Address
	return nilAddr
}

// SetMetadataFetcher sets the metadata fetcher on the block processor
func (bp *BlockProcessor) SetMetadataFetcher(fetcher MetadataFetcher) {
	// Set the fetcher on our processors
	bp.metadataFetcher = fetcher
	if bp.certificateProcessor != nil {
		bp.certificateProcessor.SetMetadataFetcher(fetcher)
	}
	if bp.governanceProcessor != nil {
		bp.governanceProcessor.SetMetadataFetcher(fetcher)
	}
}

func stringPtr(s string) *string {
	return &s
}

// isZeroHash checks if a Blake2b256 hash is all zeros
func isZeroHash(hash [32]byte) bool {
	for _, b := range hash {
		if b != 0 {
			return false
		}
	}
	return true
}

// GetDB returns the database connection
func (bp *BlockProcessor) GetDB() *gorm.DB {
	return bp.db
}

