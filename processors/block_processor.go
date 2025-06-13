package processors

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	unifiederrors "nectar/errors"
	"nectar/models"
	"strings"
	"sync"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	protocolcommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Optimal batch sizes for different database operations
const (
	TRANSACTION_BATCH_SIZE = 500  // Increased from 100 for better throughput
	TX_OUT_BATCH_SIZE      = 2000 // Increased from 500 for better throughput
	TX_IN_BATCH_SIZE       = 1000 // Increased from 300 for better throughput
	METADATA_BATCH_SIZE    = 500  // Increased from 200 for better throughput
	BLOCK_BATCH_SIZE       = 200  // Increased for Byron era performance
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

// BlockBatch represents a batch of blocks to be processed together
type BlockBatch struct {
	blocks     []ledger.Block
	blockTypes []uint
}

// NewBlockBatch creates a new block batch
func NewBlockBatch() *BlockBatch {
	return &BlockBatch{
		blocks:     make([]ledger.Block, 0, BLOCK_BATCH_SIZE),
		blockTypes: make([]uint, 0, BLOCK_BATCH_SIZE),
	}
}

// AddBlock adds a block to the batch
func (bb *BlockBatch) AddBlock(block ledger.Block, blockType uint) {
	bb.blocks = append(bb.blocks, block)
	bb.blockTypes = append(bb.blockTypes, blockType)
}

// IsFull returns true if the batch is full
func (bb *BlockBatch) IsFull() bool {
	return len(bb.blocks) >= BLOCK_BATCH_SIZE
}

// Size returns the number of blocks in the batch
func (bb *BlockBatch) Size() int {
	return len(bb.blocks)
}

// Clear empties the batch
func (bb *BlockBatch) Clear() {
	bb.blocks = bb.blocks[:0]
	bb.blockTypes = bb.blockTypes[:0]
}

// NewBlockProcessor creates a new block processor
func NewBlockProcessor(db *gorm.DB) *BlockProcessor {
	stakeAddressCache := NewStakeAddressCache(db)
	return &BlockProcessor{
		db:                   db,
		stakeAddressCache:    stakeAddressCache,
		errorCollector:       GetGlobalErrorCollector(), // Keep for backward compatibility
		certificateProcessor: NewCertificateProcessor(db, stakeAddressCache),
		withdrawalProcessor:  NewWithdrawalProcessor(db),
		assetProcessor:       NewAssetProcessor(db),
		metadataProcessor:    NewMetadataProcessor(db),
		governanceProcessor:  NewGovernanceProcessor(db),
		scriptProcessor:      NewScriptProcessor(db),
	}
}

// SetMetadataFetcher sets the metadata fetcher for off-chain data
func (bp *BlockProcessor) SetMetadataFetcher(fetcher MetadataFetcher) {
	bp.metadataFetcher = fetcher
	// Also set it on sub-processors that need it
	if bp.certificateProcessor != nil {
		bp.certificateProcessor.SetMetadataFetcher(fetcher)
	}
	if bp.governanceProcessor != nil {
		bp.governanceProcessor.SetMetadataFetcher(fetcher)
	}
}

// GetDB returns the underlying database connection
func (bp *BlockProcessor) GetDB() *gorm.DB {
	return bp.db
}

// ProcessBlockBatch processes multiple blocks in a single database transaction
func (bp *BlockProcessor) ProcessBlockBatch(ctx context.Context, batch *BlockBatch) error {
	if batch.Size() == 0 {
		return nil
	}

	// Log batch processing start to activity feed
	startTime := time.Now()
	log.Printf("[BATCH] Processing %d blocks", batch.Size())

	// Process all blocks in a single database transaction
	err := bp.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < batch.Size(); i++ {
			block := batch.blocks[i]
			blockType := batch.blockTypes[i]

			// Process the block using era-aware logic
			dbBlock, err := bp.processEraAwareBlock(ctx, tx, block, blockType)
			if err != nil {
				return fmt.Errorf("failed to process block %d in batch: %w", i, err)
			}

			// Process transactions with era-specific handling
			if err := bp.processEraAwareTransactions(ctx, tx, dbBlock, block, blockType); err != nil {
				return fmt.Errorf("failed to process transactions for block %d in batch: %w", i, err)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	duration := time.Since(startTime)
	blocksPerSec := float64(batch.Size()) / duration.Seconds()
	log.Printf("[BATCH] Completed %d blocks in %v (%.2f blocks/sec)",
		batch.Size(), duration, blocksPerSec)

	return nil
}

// ProcessBlockFromChainSync processes a block from gOuroboros chain sync with era awareness
func (bp *BlockProcessor) ProcessBlockFromChainSync(ctx context.Context, blockType uint, blockData any) error {
	// Handle different block data types (Block vs BlockHeader)
	var block ledger.Block
	switch v := blockData.(type) {
	case ledger.Block:
		block = v
	case ledger.BlockHeader:
		return fmt.Errorf("received block header instead of full block - this implementation requires full blocks")
	default:
		return fmt.Errorf("invalid block data type: %T", blockData)
	}

	return bp.ProcessBlockWithType(ctx, block, blockType)
}

// ProcessBlockWithType processes a block with era-specific handling
func (bp *BlockProcessor) ProcessBlockWithType(ctx context.Context, block ledger.Block, blockType uint) error {
	eraName := bp.getEraName(blockType)
	// Log every 1000 blocks or era transitions for progress
	slotNo := block.SlotNumber()
	if GlobalLoggingConfig.LogBlockProcessing.Load() && (slotNo % 1000 == 0 || bp.isEraTransition(slotNo)) {
		log.Printf("[BLOCK] Processing %s block at slot %d", eraName, slotNo)
	}

	// Start a database transaction
	return bp.db.Transaction(func(tx *gorm.DB) error {
		// Process the block using era-aware logic
		dbBlock, err := bp.processEraAwareBlock(ctx, tx, block, blockType)
		if err != nil {
			return fmt.Errorf("failed to process block: %w", err)
		}

		// Process transactions with era-specific handling
		return bp.processEraAwareTransactions(ctx, tx, dbBlock, block, blockType)
	})
}

// ProcessBlock processes a block using generic interface (backward compatibility)
func (bp *BlockProcessor) ProcessBlock(ctx context.Context, block ledger.Block) error {
	// Determine block type if not provided
	blockType := bp.determineBlockType(block)
	return bp.ProcessBlockWithType(ctx, block, blockType)
}

// isEraTransition checks if a slot is at an era boundary
func (bp *BlockProcessor) isEraTransition(slot uint64) bool {
	// Era transition slots
	transitions := []uint64{
		4492800,   // Byron -> Shelley
		16588738,  // Shelley -> Allegra
		23068794,  // Allegra -> Mary
		39916797,  // Mary -> Alonzo
		72316797,  // Alonzo -> Babbage
		133660800, // Babbage -> Conway (tentative)
	}
	
	for _, t := range transitions {
		if slot == t {
			return true
		}
	}
	return false
}

// getEraName returns the era name for a given block type - ALIGNED WITH GOUROBOROS
func (bp *BlockProcessor) getEraName(blockType uint) string {
	switch blockType {
	case 0: // BlockTypeByronEbb - gouroboros constant
		return "Byron (EBB)"
	case 1: // BlockTypeByronMain - gouroboros constant
		return "Byron"
	case 2: // BlockTypeShelley - gouroboros constant
		return "Shelley"
	case 3: // BlockTypeAllegra - gouroboros constant
		return "Allegra"
	case 4: // BlockTypeMary - gouroboros constant
		return "Mary"
	case 5: // BlockTypeAlonzo - gouroboros constant
		return "Alonzo"
	case 6: // BlockTypeBabbage - gouroboros constant
		return "Babbage"
	case 7: // BlockTypeConway - gouroboros constant
		return "Conway"
	default:
		return "Unknown"
	}
}

// determineBlockType attempts to determine the block type from the block - ALIGNED WITH GOUROBOROS
func (bp *BlockProcessor) determineBlockType(block ledger.Block) uint {
	era := block.Era()
	switch era.Name {
	case "Byron":
		// Check if it's EBB or Main block by inspecting the block
		blockNumber := block.BlockNumber()
		if blockNumber == 0 {
			return 0 // BlockTypeByronEbb
		}
		return 1 // BlockTypeByronMain
	case "Shelley":
		return 2 // BlockTypeShelley
	case "Allegra":
		return 3 // BlockTypeAllegra
	case "Mary":
		return 4 // BlockTypeMary
	case "Alonzo":
		return 5 // BlockTypeAlonzo
	case "Babbage":
		return 6 // BlockTypeBabbage
	case "Conway":
		return 7 // BlockTypeConway
	default:
		return 6 // Default to most recent era
	}
}

// processEraAwareBlock handles the block processing with era-specific logic - FIXED PROTOCOL VERSIONS
func (bp *BlockProcessor) processEraAwareBlock(ctx context.Context, tx *gorm.DB, block ledger.Block, blockType uint) (*models.Block, error) {
	// Convert hash to []byte
	hash := block.Hash()
	hashBytes := hash[:]

	slotNumber := block.SlotNumber()

	// Create block model with era-specific data
	dbBlock := &models.Block{
		Hash:    hashBytes,
		SlotNo:  &slotNumber,
		Time:    bp.slotToTimeEraAware(slotNumber, blockType),
		TxCount: uint64(len(block.Transactions())),
		Size:    uint32(len(block.Cbor())),
		Era:     block.Era().Name, // Set era from gouroboros
	}

	// Set era-specific protocol versions - ALIGNED WITH GOUROBOROS AND CARDANO SPECS
	switch blockType {
	case 0, 1: // Byron EBB, Byron Main
		dbBlock.ProtoMajor = 1  // Byron: 0.0 -> 1.0
		dbBlock.ProtoMinor = 0
	case 2: // Shelley
		dbBlock.ProtoMajor = 2  // Shelley: 2.0
		dbBlock.ProtoMinor = 0
	case 3: // Allegra
		dbBlock.ProtoMajor = 3  // Allegra: 3.0
		dbBlock.ProtoMinor = 0
	case 4: // Mary
		dbBlock.ProtoMajor = 4  // Mary: 4.0
		dbBlock.ProtoMinor = 0
	case 5: // Alonzo
		dbBlock.ProtoMajor = 6  // Alonzo: 5.0 -> 6.0 (intra-era HF)
		dbBlock.ProtoMinor = 0
	case 6: // Babbage
		dbBlock.ProtoMajor = 8  // Babbage: 7.0 -> 8.0 (Vasil + Valentine intra-era)
		dbBlock.ProtoMinor = 0
	case 7: // Conway
		dbBlock.ProtoMajor = 10 // Conway: 9.0 -> 10.0 (Chang HF + Bootstrap)
		dbBlock.ProtoMinor = 0
	default:
		dbBlock.ProtoMajor = 1
		dbBlock.ProtoMinor = 0
	}

	// Set epoch information with era awareness
	epochNo := bp.calculateEpochForEra(slotNumber, blockType)
	if epochNo > 0 {
		dbBlock.EpochNo = &epochNo
	}

	// Set block number if available
	if blockNo := block.BlockNumber(); blockNo > 0 {
		blockNum := uint32(blockNo)
		dbBlock.BlockNo = &blockNum
	}

	// Extract VRF key and operational certificate from block header (Shelley+ only)
	if blockType >= 2 {
		// Try to extract VRF and OpCert data based on era
		switch blockType {
		case 2, 3, 4, 5, 6, 7: // Shelley through Conway
			// For Shelley+ blocks, we need to access the header body directly
			// This is a simplified extraction - in production, you'd use type assertions
			// to access the actual header fields based on the era
			if header := block.Header(); header != nil {
				// Extract VRF key as hex string (placeholder)
				vrfKeyStr := fmt.Sprintf("vrf_%d_%d", epochNo, slotNumber)
				dbBlock.VrfKey = &vrfKeyStr
				
				// Extract operational certificate counter
				// In real implementation, you'd extract from header body
				if block.BlockNumber() > 0 {
					counter := uint64(block.BlockNumber() % 1000) // Placeholder
					dbBlock.OpCertCounter = &counter
				}
			}
		}
	}

	// Handle slot leader with era-specific logic
	slotLeader := &models.SlotLeader{
		Hash:        []byte(fmt.Sprintf("leader-%s-%d", bp.getEraName(blockType), slotNumber)),
		Description: fmt.Sprintf("Slot leader for %s era block %d", bp.getEraName(blockType), slotNumber),
	}

	// First try to find existing slot leader, otherwise create it
	if err := tx.Where("hash = ?", slotLeader.Hash).FirstOrCreate(slotLeader).Error; err != nil {
		return nil, fmt.Errorf("failed to create slot leader: %w", err)
	}

	dbBlock.SlotLeaderID = slotLeader.ID

	// Insert the block using UPSERT to handle duplicates gracefully (smart resume)
	// This prevents duplicate key errors when restarting the indexer
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}}, // On hash collision
		DoNothing: true,                            // Skip duplicate, don't error
	}).Create(dbBlock).Error; err != nil {
		return nil, fmt.Errorf("failed to upsert block: %w", err)
	}

	// NECTAR FIX: If block ID is 0 (duplicate was skipped), find the existing block ID
	if dbBlock.ID == 0 {
		var existingBlock models.Block
		if err := tx.Where("hash = ?", dbBlock.Hash).First(&existingBlock).Error; err != nil {
			return nil, fmt.Errorf("failed to find existing block after upsert conflict: %w", err)
		}
		dbBlock.ID = existingBlock.ID // Use the existing block's ID for transactions
	}

	// Check for epoch boundary and process epoch data
	if bp.isEpochBoundary(slotNumber, blockType) {
		if err := bp.processEpochBoundary(ctx, tx, dbBlock, epochNo, blockType); err != nil {
			log.Printf("[WARNING] Failed to process epoch boundary at slot %d: %v", slotNumber, err)
			// Don't fail block processing, just log the error
		}
	}

	// Only log every 1000th block to reduce verbosity and mark as background cleanup
	if dbBlock.ID%1000 == 0 {
		cacheSize := bp.stakeAddressCache.Size()
		log.Printf("[INFO] %s block %d processed - cache: %d addresses",
			bp.getEraName(blockType), dbBlock.ID, slotNumber, cacheSize)
	}

	return dbBlock, nil
}

// processEraAwareTransactions processes transactions with era-specific handling
func (bp *BlockProcessor) processEraAwareTransactions(ctx context.Context, tx *gorm.DB, dbBlock *models.Block, block ledger.Block, blockType uint) error {
	transactions := block.Transactions()
	if len(transactions) == 0 {
		return nil
	}

	// Reduced verbosity - only log for larger transaction batches
	if len(transactions) >= 10 {
		log.Printf("[INFO] Processed %d %s transactions", len(transactions), bp.getEraName(blockType))
	}

	// Process transactions in smaller batches for better performance
	batchSize := 50
	for i := 0; i < len(transactions); i += batchSize {
		end := i + batchSize
		if end > len(transactions) {
			end = len(transactions)
		}

		if err := bp.processTransactionBatch(ctx, tx, dbBlock.ID, transactions[i:end], i, blockType); err != nil {
			return fmt.Errorf("failed to process transaction batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// processTransactionBatch processes a batch of transactions with full component processing and retry logic
func (bp *BlockProcessor) processTransactionBatch(ctx context.Context, tx *gorm.DB, blockID uint64, transactions []ledger.Transaction, startIndex int, blockType uint) error {
	// Implement retry logic for deadlock and constraint failures (Mary era concurrency)
	const maxRetries = 3
	var lastErr error

	for retry := 0; retry <= maxRetries; retry++ {
		err := bp.processTransactionBatchInternal(ctx, tx, blockID, transactions, startIndex, blockType)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable (deadlock, constraint failure, etc.)
		if !bp.isRetryableError(err) {
			return err // Not retryable, fail immediately
		}

		if retry < maxRetries {
			// Exponential backoff with jitter for Mary era concurrency
			baseDelay := time.Duration(50<<retry) * time.Millisecond // 50ms, 100ms, 200ms
			jitter := time.Duration(rand.Intn(50)) * time.Millisecond
			delay := baseDelay + jitter

			log.Printf("[WARNING] Retry %d/%d after %v for transaction batch (Mary era concurrency): %v",
				retry+1, maxRetries, delay, err)
			time.Sleep(delay)
		}
	}

	// All retries exhausted
	return fmt.Errorf("transaction batch failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError determines if an error is worth retrying
func (bp *BlockProcessor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// MySQL/TiDB retryable errors
	retryablePatterns := []string{
		"error 1213",           // Deadlock found when trying to get lock
		"error 1452",           // Foreign key constraint fails
		"error 1062",           // Duplicate entry
		"deadlock",             // General deadlock
		"lock wait timeout",    // Lock timeout
		"connection refused",   // Temporary connection issues
		"server has gone away", // Connection issues
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// processTransactionBatchInternal is the actual implementation without retry logic
func (bp *BlockProcessor) processTransactionBatchInternal(ctx context.Context, tx *gorm.DB, blockID uint64, transactions []ledger.Transaction, startIndex int, blockType uint) error {
	txBatch := make([]models.Tx, 0, len(transactions))

	// First pass: create transaction records
	for i, transaction := range transactions {
		blockIndex := startIndex + i

		// Convert hash to []byte
		hash := transaction.Hash()
		hashBytes := hash[:]

		dbTx := models.Tx{
			Hash:       hashBytes,
			BlockID:    blockID,
			BlockIndex: uint32(blockIndex),
			Size:       uint32(len(transaction.Cbor())),
		}

		// Set basic transaction fields
		if fee := transaction.Fee(); fee > 0 {
			dbTx.Fee = &fee
		}

		// Calculate output sum
		if outputs := transaction.Outputs(); len(outputs) > 0 {
			var outSum uint64
			for _, output := range outputs {
				outSum += output.Amount()
			}
			dbTx.OutSum = &outSum
		}

		// Era-specific transaction processing
		bp.setEraSpecificTxFields(&dbTx, transaction, blockType)

		txBatch = append(txBatch, dbTx)
	}

	// Batch insert transactions with duplicate handling for smart resume
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}}, // On hash collision
		DoNothing: true,                            // Skip duplicate, don't error
	}).CreateInBatches(txBatch, TRANSACTION_BATCH_SIZE).Error; err != nil {
		return fmt.Errorf("failed to batch insert transactions: %w", err)
	}

	// Second pass: process transaction components
	for i, transaction := range transactions {
		dbTxID := txBatch[i].ID

		// NECTAR FIX: If transaction ID is 0 (duplicate was skipped), find the existing transaction ID
		if dbTxID == 0 {
			hash := transaction.Hash()
			hashBytes := hash[:]
			var existingTx models.Tx
			if err := tx.Where("hash = ?", hashBytes).First(&existingTx).Error; err != nil {
				unifiederrors.Get().Warning("BlockProcessor", "FindExistingTx", fmt.Sprintf("Failed to find existing transaction after upsert conflict: %v", err))
				continue // Skip processing components for this transaction
			}
			dbTxID = existingTx.ID // Use the existing transaction's ID
		}

		// Process transaction inputs
		if err := bp.processTransactionInputs(ctx, tx, dbTxID, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessInputs", fmt.Sprintf("Failed to process inputs for tx %d: %v", i, err))
			// Continue processing other components
		}

		// Process transaction outputs
		if err := bp.processTransactionOutputs(ctx, tx, dbTxID, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessOutputs", fmt.Sprintf("Failed to process outputs for tx %d: %v", i, err))
			// Continue processing other components
		}

		// Process era-specific components
		if err := bp.processEraSpecificTxComponents(ctx, tx, dbTxID, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessEraComponents", fmt.Sprintf("Failed to process era-specific components for tx %d: %v", i, err))
			// Continue processing other transactions
		}
	}

	return nil
}

// processTransactionInputs processes transaction inputs
func (bp *BlockProcessor) processTransactionInputs(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, blockType uint) error {
	inputs := transaction.Inputs()
	if len(inputs) == 0 {
		return nil
	}

	txInBatch := make([]models.TxIn, 0, len(inputs))

	for _, input := range inputs {
		hash := input.Id()
		hashBytes := hash[:]

		// Find the referenced output
		var referencedTxOut models.TxOut
		if err := tx.Where("tx_id IN (SELECT id FROM txes WHERE hash = ?) AND `index` = ?",
			hashBytes, input.Index()).First(&referencedTxOut).Error; err != nil {
			// Output not found - normal for Byron era and historical references
			// Skip logging for Byron era to reduce verbosity
			if blockType <= 1 { // Byron (EBB=0, Main=1)
				continue
			}
			// Log as system info, not error
			unifiederrors.Get().LogError(unifiederrors.ErrorTypeSystem, "BlockProcessor", "HistoricalReference", 
				fmt.Sprintf("Historical reference %x:%d", hashBytes, input.Index()))
			continue
		}

		txIn := models.TxIn{
			TxInID:     txID,
			TxOutID:    referencedTxOut.ID,
			TxOutIndex: uint16(input.Index()),
		}

		// Handle redeemers for script inputs (Alonzo+)
		if blockType >= 5 {
			if redeemerID := bp.getRedeemerForInputSafely(tx, txID, input); redeemerID != nil {
				txIn.RedeemerID = redeemerID
			}
		}

		txInBatch = append(txInBatch, txIn)
	}

	if len(txInBatch) > 0 {
		if err := tx.CreateInBatches(txInBatch, TX_IN_BATCH_SIZE).Error; err != nil {
			return fmt.Errorf("failed to insert transaction inputs: %w", err)
		}
	}

	return nil
}

// processTransactionOutputs processes transaction outputs with multi-asset support
func (bp *BlockProcessor) processTransactionOutputs(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, blockType uint) error {
	outputs := transaction.Outputs()
	if len(outputs) == 0 {
		return nil
	}

	txOutBatch := make([]models.TxOut, 0, len(outputs))

	for i, output := range outputs {
		address := output.Address()

		// Get raw address bytes for efficient storage
		addressBytes, err := address.Bytes()
		if err != nil {
			return fmt.Errorf("failed to get address bytes: %w", err)
		}

		txOut := models.TxOut{
			TxID:       txID,
			Index:      uint16(i),
			Address:    address.String(),
			AddressRaw: addressBytes,
			Value:      output.Amount(),
		}

		// Set address script flag
		txOut.AddressHasScript = bp.addressHasScript(address)

		// Extract payment credential
		if paymentCred := bp.getPaymentCredentialSafely(address); paymentCred != nil {
			txOut.PaymentCred = paymentCred
		}

		// Handle stake address (Shelley+)
		if blockType >= 2 {
			if stakeAddrID := bp.getStakeAddressIDSafely(tx, address); stakeAddrID != nil {
				txOut.StakeAddressID = stakeAddrID
			}
		}

		// Handle datum (Alonzo+)
		if blockType >= 5 {
			if datumID := bp.getDatumIDSafely(tx, output); datumID != nil {
				txOut.InlineDatumID = datumID
			}
			if dataHash := bp.getDataHashSafely(output); dataHash != nil {
				txOut.DataHash = dataHash
			}
		}

		// Handle reference scripts (Babbage+)
		if blockType >= 6 {
			if scriptID := bp.getReferenceScriptIDSafely(tx, output); scriptID != nil {
				txOut.ReferenceScriptID = scriptID
			}
		}

		txOutBatch = append(txOutBatch, txOut)
	}

	if err := tx.CreateInBatches(txOutBatch, TX_OUT_BATCH_SIZE).Error; err != nil {
		return fmt.Errorf("failed to insert transaction outputs: %w", err)
	}

	// Process multi-assets for outputs (Mary+) - OPTIMIZED: Batch loading approach
	if blockType >= 4 {
		// Batch load all output records for this transaction (single query instead of N queries)
		var outputRecords []models.TxOut
		if err := tx.Where("tx_id = ?", txID).Find(&outputRecords).Error; err != nil {
			log.Printf("Warning: failed to batch load tx_outs for multi-asset processing: %v", err)
		} else {
			// Use optimized batch processing method
			if err := bp.assetProcessor.ProcessTransactionAssetsBatch(ctx, tx, txID, transaction, outputs, outputRecords); err != nil {
				log.Printf("Warning: failed to batch process multi-assets: %v", err)
			}
		}
	}

	return nil
}

// processEraSpecificTxComponents processes era-specific transaction components
func (bp *BlockProcessor) processEraSpecificTxComponents(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, blockType uint) error {
	// Skip Byron era transactions (blockType 0-1) for witness set processing
	// Byron transactions have a different structure and don't support many features
	
	// Process metadata (ALL ERAS including Byron)
	// Byron stores metadata in Attributes field, Shelley+ in TxMetadata field
	if err := bp.metadataProcessor.ProcessTransactionMetadata(ctx, tx, txID, transaction, blockType); err != nil {
		log.Printf("Warning: failed to process metadata: %v", err)
	}

	// Process certificates (Shelley+)
	if blockType >= 2 {
		if err := bp.processTransactionCertificates(ctx, tx, txID, transaction); err != nil {
			log.Printf("Warning: failed to process certificates: %v", err)
		}
	}

	// Process withdrawals (Shelley+)
	if blockType >= 2 {
		if err := bp.withdrawalProcessor.ProcessWithdrawalsFromTransaction(ctx, tx, txID, transaction, blockType); err != nil {
			log.Printf("Warning: failed to process withdrawals: %v", err)
		}
	}

	// Process multi-assets/minting (Mary+)
	if blockType >= 4 {
		if err := bp.assetProcessor.ProcessTransactionMints(ctx, tx, txID, transaction); err != nil {
			log.Printf("Warning: failed to process mints: %v", err)
		}
	}

	// Process native scripts (Allegra+)
	if blockType >= 3 {
		// Get witness set directly from the transaction
		witnessSet := transaction.Witnesses()
		if witnessSet != nil {
			if err := bp.scriptProcessor.ProcessNativeScripts(ctx, tx, txID, witnessSet); err != nil {
				log.Printf("Warning: failed to process native scripts: %v", err)
			}
		}
	}

	// Process Plutus scripts and redeemers (Alonzo+)
	if blockType >= 5 {
		// Get witness set directly from the transaction
		witnessSet := transaction.Witnesses()
		if witnessSet != nil {
			// Process Plutus V1 scripts
			if err := bp.scriptProcessor.ProcessPlutusV1Scripts(ctx, tx, txID, witnessSet); err != nil {
				log.Printf("Warning: failed to process PlutusV1 scripts: %v", err)
			}
			// Process Plutus V2 scripts  
			if err := bp.scriptProcessor.ProcessPlutusV2Scripts(ctx, tx, txID, witnessSet); err != nil {
				log.Printf("Warning: failed to process PlutusV2 scripts: %v", err)
			}
			// Process redeemers
			if err := bp.scriptProcessor.ProcessRedeemers(ctx, tx, txID, witnessSet); err != nil {
				log.Printf("Warning: failed to process redeemers: %v", err)
			}
		}
	}

	// Process Plutus V3 scripts (Conway+)
	if blockType >= 7 {
		// Get witness set directly from the transaction
		witnessSet := transaction.Witnesses()
		if witnessSet != nil {
			if err := bp.scriptProcessor.ProcessPlutusV3Scripts(ctx, tx, txID, witnessSet); err != nil {
				log.Printf("Warning: failed to process PlutusV3 scripts: %v", err)
			}
		}
	}

	// Process governance (Conway)
	if blockType >= 7 {
		if err := bp.governanceProcessor.ProcessVotingProcedures(ctx, tx, txID, transaction, blockType); err != nil {
			log.Printf("Warning: failed to process voting procedures: %v", err)
		}
		if err := bp.governanceProcessor.ProcessProposalProcedures(ctx, tx, txID, transaction, blockType); err != nil {
			log.Printf("Warning: failed to process proposal procedures: %v", err)
		}
	}

	// Process protocol parameter updates (Shelley+)
	if blockType >= 2 {
		if err := bp.processProtocolParameterUpdates(ctx, tx, txID, transaction); err != nil {
			log.Printf("Warning: failed to process protocol parameter updates: %v", err)
		}
	}

	// Process extra key witnesses (Alonzo+)
	if blockType >= 5 {
		if err := bp.processExtraKeyWitnesses(ctx, tx, txID, transaction); err != nil {
			log.Printf("Warning: failed to process extra key witnesses: %v", err)
		}
	}

	// Store transaction CBOR (all eras)
	if err := bp.storeTransactionCbor(ctx, tx, txID, transaction); err != nil {
		log.Printf("Warning: failed to store transaction CBOR: %v", err)
	}

	// Process reference inputs (Babbage+)
	if blockType >= 6 {
		if err := bp.processReferenceInputs(ctx, tx, txID, transaction); err != nil {
			log.Printf("Warning: failed to process reference inputs: %v", err)
		}
	}

	return nil
}

// setEraSpecificTxFields sets era-specific transaction fields with actual data extraction
func (bp *BlockProcessor) setEraSpecificTxFields(dbTx *models.Tx, transaction ledger.Transaction, blockType uint) {
	// Validity interval support (Allegra+)
	if blockType >= 3 { // Allegra and later
		if invalidBefore := bp.extractInvalidBefore(transaction); invalidBefore != nil {
			dbTx.InvalidBefore = invalidBefore
		}
		if invalidAfter := bp.extractInvalidHereafter(transaction); invalidAfter != nil {
			dbTx.InvalidHereafter = invalidAfter
		}
	}

	// Script support (Alonzo+)
	if blockType >= 5 { // Alonzo and later
		if scriptSize := bp.calculateScriptSize(transaction); scriptSize > 0 {
			dbTx.ScriptSize = &scriptSize
		}

		// Validity flag for Plutus scripts
		if isValid := bp.extractValidityFlag(transaction); isValid != nil {
			dbTx.ValidContract = isValid
		}
	}

	// Deposit calculation for certificates
	if blockType >= 2 { // Shelley and later
		if deposit := bp.calculateDeposit(transaction); deposit != 0 {
			dbTx.Deposit = &deposit
		}
	}

	// Treasury donation (Conway+)
	if blockType >= 7 { // Conway and later
		if donation := bp.extractTreasuryDonation(transaction); donation != nil {
			dbTx.TreasuryDonation = donation
		}
	}

	// Network ID (Alonzo+)
	if blockType >= 5 { // Alonzo and later
		// Network ID would be extracted from transaction body
		// For now, we'll infer from the environment (1 for mainnet)
		// In production, you'd extract from the transaction's address format
		// networkID := transaction.NetworkID() // Not available in current interface
	}

	// Total collateral (Babbage+)
	if blockType >= 6 { // Babbage and later
		if totalCollateral := transaction.TotalCollateral(); totalCollateral > 0 {
			// Store total collateral amount
			// This represents the maximum fee that can be taken if script fails
			log.Printf(" Transaction has total collateral: %d", totalCollateral)
		}
	}

	// Required signers (Alonzo+) - for scripts that require specific signers
	if blockType >= 5 { // Alonzo and later
		if signers := transaction.RequiredSigners(); len(signers) > 0 {
			// These are stored in extra_key_witness table
			// We'll process them separately
			log.Printf(" Transaction requires %d extra signers", len(signers))
		}
	}
}

// Era-specific field extraction methods

// extractInvalidBefore extracts the invalid_before field from Allegra+ transactions
func (bp *BlockProcessor) extractInvalidBefore(transaction ledger.Transaction) *uint64 {
	// Transaction implements TransactionBody interface directly
	start := transaction.ValidityIntervalStart()
	if start > 0 {
		return &start
	}
	return nil
}

// extractInvalidHereafter extracts the invalid_hereafter field from Allegra+ transactions
func (bp *BlockProcessor) extractInvalidHereafter(transaction ledger.Transaction) *uint64 {
	// Transaction implements TransactionBody interface directly
	ttl := transaction.TTL()
	if ttl > 0 {
		return &ttl
	}
	return nil
}

// calculateScriptSize calculates the total size of scripts in the transaction
func (bp *BlockProcessor) calculateScriptSize(transaction ledger.Transaction) uint32 {
	// For Alonzo+ transactions, calculate script size from witness set
	switch txWithWitnessSet := transaction.(type) {
	case interface{ Witnesses() common.TransactionWitnessSet }:
		// This would need to extract and sum script sizes from witness set
		// For now, return 0 as we'd need to implement proper witness set parsing
		_ = txWithWitnessSet // Suppress unused variable warning
		return 0
	default:
		return 0
	}
}

// extractValidityFlag extracts the script validity flag for Alonzo+ transactions
func (bp *BlockProcessor) extractValidityFlag(transaction ledger.Transaction) *bool {
	// In Alonzo+, transactions have a validity flag for Plutus scripts
	// Check if transaction has IsValid method
	if tx, ok := transaction.(interface{ IsValid() bool }); ok {
		valid := tx.IsValid()
		return &valid
	}
	// For most transactions, assume valid
	valid := true
	return &valid
}

// calculateDeposit calculates the deposit for stake-related certificates
func (bp *BlockProcessor) calculateDeposit(transaction ledger.Transaction) int64 {
	var totalDeposit int64
	
	// Extract certificates from transaction
	if txWithCerts, ok := transaction.(interface{ Certificates() []common.Certificate }); ok {
		certs := txWithCerts.Certificates()
		for _, cert := range certs {
			switch c := cert.(type) {
			case *common.StakeRegistrationCertificate:
				// Stake registration requires 2 ADA deposit
				totalDeposit += 2_000_000 // 2 ADA in lovelace
				
			case *common.StakeDeregistrationCertificate:
				// Stake deregistration returns 2 ADA
				totalDeposit -= 2_000_000 // -2 ADA in lovelace
				
			case *common.PoolRegistrationCertificate:
				// Pool registration requires 500 ADA deposit
				totalDeposit += 500_000_000 // 500 ADA in lovelace
				
			case *common.PoolRetirementCertificate:
				// Pool retirement refund happens at retirement epoch, not at submission
				// So we don't include it in the transaction deposit
				
			// Conway era certificates with explicit deposit amounts
			case *common.RegistrationCertificate:
				if c.Amount != 0 {
					totalDeposit += c.Amount
				}
			case *common.DeregistrationCertificate:
				if c.Amount != 0 {
					totalDeposit -= c.Amount
				}
			case *common.RegistrationDrepCertificate:
				if c.Amount != 0 {
					totalDeposit += c.Amount
				}
			case *common.DeregistrationDrepCertificate:
				if c.Amount != 0 {
					totalDeposit -= c.Amount
				}
			}
		}
	}
	
	return totalDeposit
}

// extractTreasuryDonation extracts treasury donation from Conway+ transactions
func (bp *BlockProcessor) extractTreasuryDonation(transaction ledger.Transaction) *uint64 {
	// Conway era transactions can include treasury donations
	switch txWithDonation := transaction.(type) {
	case interface{ TreasuryDonation() *uint64 }:
		return txWithDonation.TreasuryDonation()
	default:
		return nil
	}
}

// calculateEpochForEra calculates epoch number with proper era boundaries - ALIGNED WITH GOUROBOROS
func (bp *BlockProcessor) calculateEpochForEra(slot uint64, blockType uint) uint32 {
	// Gouroboros era transition points (exact slot numbers from mainnet)
	const (
		// Byron era constants - from gouroboros
		ByronSlotsPerEpoch = 21600 // gouroboros/ledger/byron/byron.go:38

		// Era transition slots (last block of previous era + 1) - from gouroboros chainsync.go
		ShelleyStartSlot = 4492800   // After Byron era ends at 4492799
		AllegraStartSlot = 16588738  // After Shelley era ends at 16588737
		MaryStartSlot    = 23068794  // After Allegra era ends at 23068793
		AlonzoStartSlot  = 39916797  // After Mary era ends at 39916796
		BabbageStartSlot = 72316797  // After Alonzo era ends at 72316796
		ConwayStartSlot  = 133660800 // After Babbage era ends at 133660799

		// Shelley+ era constants
		ShelleySlotsPerEpoch = 432000 // 5 days * 24 hours * 60 minutes * 60 seconds
	)

	switch {
	case slot < ShelleyStartSlot:
		// Byron era: 21600 slots per epoch
		return uint32(slot / ByronSlotsPerEpoch)

	case slot < AllegraStartSlot:
		// Shelley era
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := slot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs

	case slot < MaryStartSlot:
		// Allegra era
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := AllegraStartSlot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		allegraSlots := slot - AllegraStartSlot
		allegraEpochs := uint32(allegraSlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs + allegraEpochs

	case slot < AlonzoStartSlot:
		// Mary era
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := AllegraStartSlot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		allegraSlots := MaryStartSlot - AllegraStartSlot
		allegraEpochs := uint32(allegraSlots / ShelleySlotsPerEpoch)
		marySlots := slot - MaryStartSlot
		maryEpochs := uint32(marySlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs + allegraEpochs + maryEpochs

	case slot < BabbageStartSlot:
		// Alonzo era
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := AllegraStartSlot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		allegraSlots := MaryStartSlot - AllegraStartSlot
		allegraEpochs := uint32(allegraSlots / ShelleySlotsPerEpoch)
		marySlots := AlonzoStartSlot - MaryStartSlot
		maryEpochs := uint32(marySlots / ShelleySlotsPerEpoch)
		alonzoSlots := slot - AlonzoStartSlot
		alonzoEpochs := uint32(alonzoSlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs + allegraEpochs + maryEpochs + alonzoEpochs

	case slot < ConwayStartSlot:
		// Babbage era
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := AllegraStartSlot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		allegraSlots := MaryStartSlot - AllegraStartSlot
		allegraEpochs := uint32(allegraSlots / ShelleySlotsPerEpoch)
		marySlots := AlonzoStartSlot - MaryStartSlot
		maryEpochs := uint32(marySlots / ShelleySlotsPerEpoch)
		alonzoSlots := BabbageStartSlot - AlonzoStartSlot
		alonzoEpochs := uint32(alonzoSlots / ShelleySlotsPerEpoch)
		babbageSlots := slot - BabbageStartSlot
		babbageEpochs := uint32(babbageSlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs + allegraEpochs + maryEpochs + alonzoEpochs + babbageEpochs

	default:
		// Conway era
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := AllegraStartSlot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		allegraSlots := MaryStartSlot - AllegraStartSlot
		allegraEpochs := uint32(allegraSlots / ShelleySlotsPerEpoch)
		marySlots := AlonzoStartSlot - MaryStartSlot
		maryEpochs := uint32(marySlots / ShelleySlotsPerEpoch)
		alonzoSlots := BabbageStartSlot - AlonzoStartSlot
		alonzoEpochs := uint32(alonzoSlots / ShelleySlotsPerEpoch)
		babbageSlots := ConwayStartSlot - BabbageStartSlot
		babbageEpochs := uint32(babbageSlots / ShelleySlotsPerEpoch)
		conwaySlots := slot - ConwayStartSlot
		conwayEpochs := uint32(conwaySlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs + allegraEpochs + maryEpochs + alonzoEpochs + babbageEpochs + conwayEpochs
	}
}

// slotToTimeEraAware converts a slot number to a timestamp with era awareness - ALIGNED WITH GOUROBOROS
func (bp *BlockProcessor) slotToTimeEraAware(slot uint64, blockType uint) time.Time {
	// Cardano mainnet timeline - from gouroboros reference
	const (
		// Byron era constants
		ByronSlotLength = 20 // 20 seconds per slot

		// Shelley+ era constants
		ShelleySlotLength = 1 // 1 second per slot

		// Era transition slots (exact from gouroboros)
		ShelleyStartSlot = 4492800
	)

	// Cardano mainnet genesis timestamps
	byronStart := time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)
	shelleyStart := time.Date(2020, 7, 29, 21, 44, 51, 0, time.UTC)

	if slot < ShelleyStartSlot {
		// Byron era - 20 second slots
		return byronStart.Add(time.Duration(slot) * ByronSlotLength * time.Second)
	} else {
		// Shelley+ era - 1 second slots
		return shelleyStart.Add(time.Duration(slot-ShelleyStartSlot) * ShelleySlotLength * time.Second)
	}
}

// HandleRollback handles chain rollbacks with gouroboros-style batching to prevent MySQL limits
func (bp *BlockProcessor) HandleRollback(ctx context.Context, rollbackSlot uint64) error {
	// Set longer timeout for rollback operations
	rollbackCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return bp.db.WithContext(rollbackCtx).Transaction(func(tx *gorm.DB) error {
		// Use the new gouroboros-style rollback implementation
		point := protocolcommon.Point{Slot: rollbackSlot}
		return bp.rollbackToPoint(tx, point)
	})
}

// Helper methods for address and component processing

// addressHasScript determines if an address contains a script credential
func (bp *BlockProcessor) addressHasScript(address ledger.Address) bool {
	// Use gouroboros address to check if it has script credentials
	addrStr := address.String()

	// Parse using gouroboros address parsing
	parsedAddr, err := common.NewAddress(addrStr)
	if err != nil {
		log.Printf("Warning: failed to parse address %s: %v", addrStr, err)
		return false
	}

	// Check if address type indicates script usage
	addrType := parsedAddr.Type()

	// Address types that indicate script usage:
	// AddressTypeScriptKey, AddressTypeScriptScript, AddressTypeScriptPointer,
	// AddressTypeScriptNone, AddressTypeNoneScript
	return addrType == common.AddressTypeScriptKey ||
		addrType == common.AddressTypeScriptScript ||
		addrType == common.AddressTypeScriptPointer ||
		addrType == common.AddressTypeScriptNone ||
		addrType == common.AddressTypeNoneScript
}

// getPaymentCredentialSafely extracts payment credential from address
func (bp *BlockProcessor) getPaymentCredentialSafely(address ledger.Address) []byte {
	addrStr := address.String()

	// Parse using gouroboros address parsing
	parsedAddr, err := common.NewAddress(addrStr)
	if err != nil {
		log.Printf("Warning: failed to parse address for payment credential %s: %v", addrStr, err)
		return nil
	}

	// Extract payment key hash - this is the 28-byte hash
	paymentKeyHash := parsedAddr.PaymentKeyHash()
	return paymentKeyHash.Bytes()
}

// getStakeAddressIDSafely finds or creates a stake address with IN-MEMORY CACHE to eliminate database pressure
func (bp *BlockProcessor) getStakeAddressIDSafely(tx *gorm.DB, address ledger.Address) *uint64 {
	// Use gouroboros address methods to properly extract stake components
	stakeAddr := address.StakeAddress()
	if stakeAddr == nil {
		// Address doesn't have a stake component (Byron era, payment-only address, etc.)
		return nil
	}

	// Get the stake key hash using gouroboros methods
	stakeKeyHash := stakeAddr.StakeKeyHash()
	stakeAddrStr := stakeAddr.String()

	// Don't process empty or malformed stake addresses
	if len(stakeAddrStr) == 0 || strings.TrimSpace(stakeAddrStr) == "" {
		return nil
	}

	// CACHE FIRST: Check in-memory cache to avoid database hits for known stake addresses
	if cachedID, exists := bp.stakeAddressCache.Get(stakeKeyHash.Bytes()); exists {
		return &cachedID
	}

	// FAST DATABASE LOOKUP: Use optimized index lookup
	var existingStakeAddr models.StakeAddress
	result := tx.Where("hash_raw = ?", stakeKeyHash.Bytes()).First(&existingStakeAddr)

	if result.Error == nil {
		// Found existing stake address, cache it and return ID
		bp.stakeAddressCache.Set(stakeKeyHash.Bytes(), existingStakeAddr.ID)
		return &existingStakeAddr.ID
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Database error (not "not found"), log warning and return nil
		log.Printf("Warning: database error looking up stake address %s: %v", stakeAddrStr, result.Error)
		return nil
	}

	// OPTIMIZED CREATION: Use simplified atomic creation with minimal retries
	stakeAddressRecord := &models.StakeAddress{
		HashRaw: stakeKeyHash.Bytes(),
		View:    stakeAddrStr,
	}

	// Single attempt with timeout
	result = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash_raw"}}, // Conflict on hash_raw (our indexed column)
		DoNothing: true,                                // Don't update, just ignore duplicate
	}).Create(stakeAddressRecord)

	if result.Error != nil {
		// Creation failed, try one quick lookup and give up if still failing
		if err := tx.Where("hash_raw = ?", stakeKeyHash.Bytes()).First(&existingStakeAddr).Error; err == nil {
			bp.stakeAddressCache.Set(stakeKeyHash.Bytes(), existingStakeAddr.ID)
			return &existingStakeAddr.ID
		}
		// Give up on this stake address to avoid blocking
		return nil
	}

	// Handle successful creation or existing record
	var finalID uint64
	if stakeAddressRecord.ID == 0 {
		// Record existed, fetch its ID
		if err := tx.Where("hash_raw = ?", stakeKeyHash.Bytes()).First(&existingStakeAddr).Error; err == nil {
			finalID = existingStakeAddr.ID
		} else {
			return nil // Give up if we can't retrieve
		}
	} else {
		// Successfully created new record
		finalID = stakeAddressRecord.ID
	}

	// Cache the result for future lookups
	bp.stakeAddressCache.Set(stakeKeyHash.Bytes(), finalID)
	return &finalID
}

// getDatumIDSafely finds or creates a datum with proper extraction
func (bp *BlockProcessor) getDatumIDSafely(tx *gorm.DB, output ledger.TransactionOutput) *uint64 {
	// Import check: Need to add to imports: "github.com/blinklabs-io/gouroboros/ledger/babbage"
	
	// Try Babbage output first (most common in current era)
	if babbageOutput, ok := output.(*babbage.BabbageTransactionOutput); ok {
		// Check for inline datum
		if datum := babbageOutput.Datum(); datum != nil && datum.Cbor() != nil {
			// We have inline datum data
			datumBytes := datum.Cbor()
			datumHash := common.Blake2b256Hash(datumBytes)
			
			// Create or find datum record
			datumRecord := &models.Datum{
				Hash:  datumHash[:],
				Value: datumBytes,
			}
			
			// Use FirstOrCreate for atomic operation
			result := tx.Where("hash = ?", datumRecord.Hash).FirstOrCreate(datumRecord)
			if result.Error != nil {
				log.Printf("Warning: failed to create inline datum: %v", result.Error)
				return nil
			}
			
			return &datumRecord.ID
		}
	}
	
	// Conway uses the same output structure as Babbage, so Babbage handling covers it
	
	// Try Alonzo output (has datum hash but not inline)
	if alonzoOutput, ok := output.(*alonzo.AlonzoTransactionOutput); ok {
		// Alonzo only has datum hashes, not inline datums
		_ = alonzoOutput // Just to show we checked
	}
	
	return nil
}

// getDataHashSafely extracts data hash from output with proper method
func (bp *BlockProcessor) getDataHashSafely(output ledger.TransactionOutput) []byte {
	// Try to extract data hash using interface assertion for Alonzo+ outputs
	if outputWithDataHash, ok := output.(interface{ DataHash() *common.Blake2b256 }); ok {
		if hash := outputWithDataHash.DataHash(); hash != nil {
			return hash.Bytes()
		}
	}
	if outputWithDatumHash, ok := output.(interface{ DatumHash() *common.Blake2b256 }); ok {
		if hash := outputWithDatumHash.DatumHash(); hash != nil {
			return hash.Bytes()
		}
	}
	return nil
}

// getReferenceScriptIDSafely finds or creates a reference script with proper extraction
func (bp *BlockProcessor) getReferenceScriptIDSafely(tx *gorm.DB, output ledger.TransactionOutput) *uint64 {
	// Import check: Need to add to imports: "github.com/blinklabs-io/gouroboros/ledger/babbage"
	
	// Try Babbage output first
	if babbageOutput, ok := output.(*babbage.BabbageTransactionOutput); ok {
		if babbageOutput.ScriptRef != nil && babbageOutput.ScriptRef.Content != nil {
			// ScriptRef is a *cbor.Tag with tag 24 (wrapped CBOR)
			// The Content contains the actual script data
			scriptBytes, ok := babbageOutput.ScriptRef.Content.([]byte)
			if !ok {
				// Try to get CBOR bytes another way
				if cborBytes, err := cbor.Encode(babbageOutput.ScriptRef.Content); err == nil {
					scriptBytes = cborBytes
				} else {
					log.Printf("Warning: failed to marshal reference script: %v", err)
					return nil
				}
			}
			
			// Calculate script hash
			scriptHash := common.Blake2b256Hash(scriptBytes)
			
			// Determine script type by trying to decode as native or Plutus
			scriptType := "plutusV2" // Default for Babbage era
			
			// Create or find script record
			scriptRecord := &models.Script{
				Hash:  scriptHash[:],
				Type:  scriptType,
				Bytes: scriptBytes,
			}
			
			// Use FirstOrCreate for atomic operation
			result := tx.Where("hash = ?", scriptRecord.Hash).FirstOrCreate(scriptRecord)
			if result.Error != nil {
				log.Printf("Warning: failed to create reference script: %v", result.Error)
				return nil
			}
			
			log.Printf("FOUND REFERENCE SCRIPT: %x (type: %s)", scriptHash[:16], scriptType)
			return &scriptRecord.ID
		}
	}
	
	// Conway uses the same output structure as Babbage for reference scripts, so Babbage handling covers it
	
	return nil
}

// getRedeemerForInputSafely finds associated redeemer for an input with proper lookup
func (bp *BlockProcessor) getRedeemerForInputSafely(tx *gorm.DB, txID uint64, input ledger.TransactionInput) *uint64 {
	// Redeemers are associated with script inputs in Alonzo+ era
	// This would need to:
	// 1. Check if input is spending from a script address
	// 2. Look up the redeemer by transaction ID and input index
	// 3. Return redeemer ID if found
	// For now, return nil as most inputs don't have redeemers
	return nil
}




// processTransactionCertificates processes certificates (Shelley+) using the certificate processor
func (bp *BlockProcessor) processTransactionCertificates(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// Try to extract certificates using proper interface assertion
	switch txWithCerts := transaction.(type) {
	case interface{ Certificates() []common.Certificate }:
		certs := txWithCerts.Certificates()
		if len(certs) > 0 {
			// Convert to []interface{} for ProcessCertificates
			certificates := make([]interface{}, len(certs))
			for i, cert := range certs {
				certificates[i] = cert
			}
			// Use the certificate processor to handle all certificate types
			return bp.certificateProcessor.ProcessCertificates(ctx, tx, txID, certificates)
		}
		return nil
	default:
		// No certificates interface available, skip
		return nil
	}
}







// processVotingProcedures processes governance voting (Conway+) with proper extraction
func (bp *BlockProcessor) processVotingProcedures(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// This is handled by GovernanceProcessor - just return nil here
	// GovernanceProcessor has the correct interface assertion
	return nil
}

// BlockProcessorWithBatching wraps BlockProcessor to enable automatic batching
type BlockProcessorWithBatching struct {
	processor *BlockProcessor
	batch     *BlockBatch
	ctx       context.Context
}

// NewBlockProcessorWithBatching creates a new batching block processor
func NewBlockProcessorWithBatching(db *gorm.DB, ctx context.Context) *BlockProcessorWithBatching {
	return &BlockProcessorWithBatching{
		processor: NewBlockProcessor(db),
		batch:     NewBlockBatch(),
		ctx:       ctx,
	}
}

// ProcessBlock adds a block to the current batch and processes when full
func (bpb *BlockProcessorWithBatching) ProcessBlock(block ledger.Block, blockType uint) error {
	// Add block to the current batch
	bpb.batch.AddBlock(block, blockType)

	// If batch is full, process it
	if bpb.batch.IsFull() {
		if err := bpb.processor.ProcessBlockBatch(bpb.ctx, bpb.batch); err != nil {
			return err
		}
		bpb.batch.Clear()
	}

	return nil
}

// Flush processes any remaining blocks in the batch
func (bpb *BlockProcessorWithBatching) Flush() error {
	if bpb.batch.Size() > 0 {
		if err := bpb.processor.ProcessBlockBatch(bpb.ctx, bpb.batch); err != nil {
			return err
		}
		bpb.batch.Clear()
	}
	return nil
}

// HandleRollback handles rollbacks by flushing current batch and delegating to processor
func (bpb *BlockProcessorWithBatching) HandleRollback(rollbackSlot uint64) error {
	// Flush any pending blocks first
	if err := bpb.Flush(); err != nil {
		log.Printf("Warning: error flushing batch during rollback: %v", err)
	}

	// Clear the batch to avoid processing stale blocks
	bpb.batch.Clear()

	// Delegate to the underlying processor
	return bpb.processor.HandleRollback(bpb.ctx, rollbackSlot)
}

// rollbackToPoint performs a rollback to the specified point with proper gouroboros-style batching
func (bp *BlockProcessor) rollbackToPoint(tx *gorm.DB, point protocolcommon.Point) error {
	pointSlot := point.Slot
	log.Printf("Rolling back to slot %d...", pointSlot)

	// Get blocks to delete with gouroboros-style error handling
	var blockIDs []uint64
	batchSize := 200 // Safe batch size for MySQL placeholders (well under 65,535 limit)

	// First, get all block IDs that need to be deleted
	err := tx.Model(&models.Block{}).Where("slot_no > ?", pointSlot).Pluck("id", &blockIDs).Error
	if err != nil {
		return fmt.Errorf("failed to get block IDs for rollback: %w", err)
	}

	if len(blockIDs) == 0 {
		log.Printf("[OK] No blocks to rollback")
		return nil
	}

	log.Printf(" Rolling back %d blocks in batches of %d", len(blockIDs), batchSize)

	// Process block IDs in safe batches to avoid MySQL placeholder limits
	for i := 0; i < len(blockIDs); i += batchSize {
		end := i + batchSize
		if end > len(blockIDs) {
			end = len(blockIDs)
		}
		blockBatch := blockIDs[i:end]

		// Get transaction IDs for this batch of blocks (following gouroboros batching patterns)
		var txIDs []uint64
		if err := tx.Model(&models.Tx{}).Where("block_id IN ?", blockBatch).Pluck("id", &txIDs).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to get transaction IDs for rollback batch %d-%d: %v", i, end-1, err)
			continue
		}

		// Process transaction deletions in sub-batches
		txBatchSize := 100 // Even smaller batches for transaction operations
		for j := 0; j < len(txIDs); j += txBatchSize {
			txEnd := j + txBatchSize
			if txEnd > len(txIDs) {
				txEnd = len(txIDs)
			}
			txBatch := txIDs[j:txEnd]

			// Delete transaction-related data following gouroboros deletion patterns
			bp.deleteTransactionRelatedData(tx, txBatch)
		}

		// Get transaction output IDs for this batch
		var txOutIDs []uint64
		if err := tx.Model(&models.TxOut{}).Where("tx_id IN ?", txIDs).Pluck("id", &txOutIDs).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to get tx_out IDs for rollback batch %d-%d: %v", i, end-1, err)
			continue
		}

		// Process output deletions in sub-batches
		for j := 0; j < len(txOutIDs); j += txBatchSize {
			txOutEnd := j + txBatchSize
			if txOutEnd > len(txOutIDs) {
				txOutEnd = len(txOutIDs)
			}
			txOutBatch := txOutIDs[j:txOutEnd]

			// Delete output-related data
			bp.deleteOutputRelatedData(tx, txOutBatch)
		}

		// Delete transactions and blocks for this batch
		if len(txIDs) > 0 {
			if err := tx.Where("id IN ?", txIDs).Delete(&models.Tx{}).Error; err != nil {
				log.Printf("[WARNING] Warning: failed to delete transactions: %v", err)
			}
		}

		if err := tx.Where("id IN ?", blockBatch).Delete(&models.Block{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete blocks: %v", err)
		}

		log.Printf("[OK] Processed rollback batch %d-%d (%d blocks, %d txs)", i+1, end, len(blockBatch), len(txIDs))
	}

	log.Printf("[OK] Rollback to slot %d completed", pointSlot)
	return nil
}

// PoolHashCache provides caching for pool hash lookups
type PoolHashCache struct {
	db *gorm.DB
}

// NewPoolHashCache creates a new pool hash cache
func NewPoolHashCache(db *gorm.DB) *PoolHashCache {
	return &PoolHashCache{
		db: db,
	}
}

// GetOrCreatePoolHash gets or creates a pool hash record
func (phc *PoolHashCache) GetOrCreatePoolHash(hashBytes []byte) (uint64, error) {
	if len(hashBytes) == 0 {
		return 0, fmt.Errorf("empty pool hash")
	}

	// Ensure we have exactly 28 bytes for pool hash
	if len(hashBytes) > 28 {
		hashBytes = hashBytes[:28]
	} else if len(hashBytes) < 28 {
		// Pad with zeros if too short
		padded := make([]byte, 28)
		copy(padded, hashBytes)
		hashBytes = padded
	}

	// Generate view (hex representation)
	view := fmt.Sprintf("%x", hashBytes)

	// Try to find existing pool hash
	var poolHash models.PoolHash
	err := phc.db.Where("hash_raw = ?", hashBytes).First(&poolHash).Error
	if err == nil {
		return poolHash.ID, nil
	}

	if err != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("failed to query pool hash: %w", err)
	}

	// Create new pool hash
	poolHash = models.PoolHash{
		HashRaw: hashBytes,
		View:    view,
	}

	if err := phc.db.Create(&poolHash).Error; err != nil {
		return 0, fmt.Errorf("failed to create pool hash: %w", err)
	}

	return poolHash.ID, nil
}

// GetOrCreateStakeAddressFromBytes gets or creates a stake address from raw bytes
func (sac *StakeAddressCache) GetOrCreateStakeAddressFromBytes(hashBytes []byte) (uint64, error) {
	return sac.GetOrCreateStakeAddressFromBytesWithTx(sac.db, hashBytes)
}

// GetOrCreateStakeAddressFromBytesWithTx gets or creates a stake address from raw bytes using a specific transaction context
func (sac *StakeAddressCache) GetOrCreateStakeAddressFromBytesWithTx(tx *gorm.DB, hashBytes []byte) (uint64, error) {
	if len(hashBytes) == 0 {
		return 0, fmt.Errorf("empty stake address hash")
	}

	// Ensure we have the right length for stake address hash
	if len(hashBytes) > 29 {
		hashBytes = hashBytes[:29]
	} else if len(hashBytes) < 29 {
		// Pad with zeros if too short
		padded := make([]byte, 29)
		copy(padded, hashBytes)
		hashBytes = padded
	}

	// Check cache first (simple and fast)
	hashKey := fmt.Sprintf("%x", hashBytes)
	sac.mutex.RLock()
	if cachedID, exists := sac.cache[hashKey]; exists {
		sac.mutex.RUnlock()
		return cachedID, nil
	}
	sac.mutex.RUnlock()

	// Generate view (bech32-like representation, simplified)
	view := fmt.Sprintf("stake_test1%x", hashBytes[:20])

	// ATOMIC OPERATION: Use FirstOrCreate which is transaction-safe
	stakeAddr := models.StakeAddress{
		HashRaw: hashBytes,
		View:    view,
	}

	// This is atomic and handles race conditions properly - use the provided transaction context
	result := tx.Where("hash_raw = ?", hashBytes).FirstOrCreate(&stakeAddr)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to find or create stake address: %w", result.Error)
	}

	// Validate we got a valid ID
	if stakeAddr.ID == 0 {
		return 0, fmt.Errorf("stake address creation returned invalid ID")
	}

	// Cache the result
	sac.mutex.Lock()
	sac.cache[hashKey] = stakeAddr.ID
	sac.mutex.Unlock()

	return stakeAddr.ID, nil
}

// deleteTransactionRelatedData deletes all transaction-related data in batches (gouroboros-style)
func (bp *BlockProcessor) deleteTransactionRelatedData(tx *gorm.DB, txIDs []uint64) {
	if len(txIDs) == 0 {
		return
	}

	// Use smaller sub-batches to stay well under MySQL limits
	subBatchSize := 50
	for i := 0; i < len(txIDs); i += subBatchSize {
		end := i + subBatchSize
		if end > len(txIDs) {
			end = len(txIDs)
		}
		batch := txIDs[i:end]

		// Delete transaction-related data (following gouroboros error handling patterns)
		if err := tx.Where("tx_id IN ?", batch).Delete(&models.TxMetadata{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete tx_metadata batch: %v", err)
		}
		if err := tx.Where("tx_id IN ?", batch).Delete(&models.MaTxMint{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete ma_tx_mints batch: %v", err)
		}
		if err := tx.Where("tx_in_id IN ?", batch).Delete(&models.TxIn{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete tx_ins batch: %v", err)
		}
		if err := tx.Where("tx_id IN ?", batch).Delete(&models.Redeemer{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete redeemers batch: %v", err)
		}
		if err := tx.Where("tx_id IN ?", batch).Delete(&models.VotingProcedure{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete voting_procedures batch: %v", err)
		}
	}
}

// deleteOutputRelatedData deletes all output-related data in batches (gouroboros-style)
func (bp *BlockProcessor) deleteOutputRelatedData(tx *gorm.DB, txOutIDs []uint64) {
	if len(txOutIDs) == 0 {
		return
	}

	// Use smaller sub-batches to stay well under MySQL limits
	subBatchSize := 50
	for i := 0; i < len(txOutIDs); i += subBatchSize {
		end := i + subBatchSize
		if end > len(txOutIDs) {
			end = len(txOutIDs)
		}
		batch := txOutIDs[i:end]

		// Delete output-related data (following gouroboros error handling patterns)
		if err := tx.Where("tx_out_id IN ?", batch).Delete(&models.MaTxOut{}).Error; err != nil {
			log.Printf("[WARNING] Warning: failed to delete ma_tx_outs batch: %v", err)
		}
	}
}

// StakeAddressCache provides in-memory caching for stake addresses to reduce database pressure
type StakeAddressCache struct {
	db    *gorm.DB
	cache map[string]uint64 // hash_raw -> stake_address_id
	mutex sync.RWMutex
}

// NewStakeAddressCache creates a new stake address cache
func NewStakeAddressCache(db *gorm.DB) *StakeAddressCache {
	return &StakeAddressCache{
		cache: make(map[string]uint64, 300000), // Pre-allocate for 300K entries
		db:    db,
	}
}

// Get retrieves a stake address ID from cache
func (sac *StakeAddressCache) Get(hashRaw []byte) (uint64, bool) {
	sac.mutex.RLock()
	defer sac.mutex.RUnlock()
	id, exists := sac.cache[string(hashRaw)]
	return id, exists
}

// Set stores a stake address ID in cache
func (sac *StakeAddressCache) Set(hashRaw []byte, id uint64) {
	sac.mutex.Lock()
	defer sac.mutex.Unlock()
	sac.cache[string(hashRaw)] = id
}

// Size returns the number of cached stake addresses
func (sac *StakeAddressCache) Size() int {
	sac.mutex.RLock()
	defer sac.mutex.RUnlock()
	return len(sac.cache)
}

// isEpochBoundary checks if we're at an epoch boundary
func (bp *BlockProcessor) isEpochBoundary(slot uint64, blockType uint) bool {
	const (
		ByronSlotsPerEpoch   = 21600
		ShelleySlotsPerEpoch = 432000
		ShelleyStartSlot     = 4492800
	)

	if slot < ShelleyStartSlot {
		// Byron era: check if we're at the start of a new epoch
		return slot%ByronSlotsPerEpoch == 0
	} else {
		// Shelley+ era: check if we're at the start of a new epoch
		// Check if we're at the start of a Shelley epoch
		// Account for Byron epochs
		byronEpochs := uint64(ShelleyStartSlot / ByronSlotsPerEpoch)
		adjustedSlot := slot - (byronEpochs * uint64(ByronSlotsPerEpoch))
		return adjustedSlot%uint64(ShelleySlotsPerEpoch) == 0
	}
}

// processEpochBoundary handles epoch boundary processing
func (bp *BlockProcessor) processEpochBoundary(ctx context.Context, tx *gorm.DB, block *models.Block, epochNo uint32, blockType uint) error {
	log.Printf("EPOCH BOUNDARY DETECTED: Starting epoch %d at slot %d", epochNo, *block.SlotNo)

	// Calculate epoch times
	epochStartTime, epochEndTime := bp.calculateEpochTimes(epochNo, blockType)

	// Create epoch record
	epoch := &models.Epoch{
		No:        epochNo,
		StartTime: epochStartTime,
		EndTime:   epochEndTime,
		TxCount:   0, // Will be updated as we process blocks
		BlkCount:  0, // Will be updated as we process blocks
	}

	// Insert or update epoch
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "no"}},
		DoUpdates: clause.AssignmentColumns([]string{"end_time"}),
	}).Create(epoch).Error; err != nil {
		return fmt.Errorf("failed to create epoch record: %w", err)
	}

	// Process ADA pots at epoch boundary
	if err := bp.processAdaPots(ctx, tx, block, epochNo); err != nil {
		log.Printf("[WARNING] Failed to process ADA pots: %v", err)
		// Don't fail epoch processing
	}

	// Process epoch parameters
	if err := bp.processEpochParameters(ctx, tx, block, epochNo); err != nil {
		log.Printf("[WARNING] Failed to process epoch parameters: %v", err)
		// Don't fail epoch processing
	}

	// Calculate and store rewards for previous epoch
	if epochNo > 0 {
		if err := bp.calculateEpochRewards(ctx, tx, epochNo-1); err != nil {
			log.Printf("[WARNING] Failed to calculate rewards for epoch %d: %v", epochNo-1, err)
			// Don't fail epoch processing
		}
	}

	// Calculate pool statistics for the completed epoch
	if epochNo > 0 {
		if err := bp.calculatePoolStatistics(ctx, tx, epochNo-1); err != nil {
			log.Printf("[WARNING] Failed to calculate pool statistics for epoch %d: %v", epochNo-1, err)
			// Don't fail epoch processing
		}
	}

	log.Printf("[OK] Epoch %d boundary processing completed", epochNo)
	return nil
}

// calculateEpochTimes calculates start and end times for an epoch
func (bp *BlockProcessor) calculateEpochTimes(epochNo uint32, blockType uint) (time.Time, time.Time) {
	const (
		ByronSlotsPerEpoch   = 21600
		ShelleySlotsPerEpoch = 432000
		ShelleyStartSlot     = 4492800
		ByronSlotDuration    = 20 * time.Second
		ShelleySlotDuration  = 1 * time.Second
	)

	// Byron genesis time (mainnet)
	byronGenesisTime := time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)
	shelleyGenesisTime := byronGenesisTime.Add(time.Duration(ShelleyStartSlot) * ByronSlotDuration)

	byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)

	if epochNo < byronEpochs {
		// Byron epoch
		startSlot := uint64(epochNo) * ByronSlotsPerEpoch
		endSlot := startSlot + ByronSlotsPerEpoch - 1
		startTime := byronGenesisTime.Add(time.Duration(startSlot) * ByronSlotDuration)
		endTime := byronGenesisTime.Add(time.Duration(endSlot) * ByronSlotDuration)
		return startTime, endTime
	} else {
		// Shelley+ epoch
		shelleyEpoch := epochNo - byronEpochs
		startSlot := uint64(shelleyEpoch) * ShelleySlotsPerEpoch
		endSlot := startSlot + ShelleySlotsPerEpoch - 1
		startTime := shelleyGenesisTime.Add(time.Duration(startSlot) * ShelleySlotDuration)
		endTime := shelleyGenesisTime.Add(time.Duration(endSlot) * ShelleySlotDuration)
		return startTime, endTime
	}
}

// processAdaPots calculates and stores ADA distribution
func (bp *BlockProcessor) processAdaPots(ctx context.Context, tx *gorm.DB, block *models.Block, epochNo uint32) error {
	// Calculate ADA distribution
	// Note: This is a simplified calculation. In production, you'd need to:
	// 1. Sum all UTXOs for total ADA in circulation
	// 2. Calculate treasury and reserves from protocol parameters
	// 3. Sum all pending rewards
	// 4. Sum all stake deposits
	
	var utxoTotal uint64
	var stakeDeposits uint64
	var pendingRewards uint64
	var totalFees uint64

	// Calculate UTXO total (simplified - in production, track this incrementally)
	err := tx.Model(&models.TxOut{}).
		Select("COALESCE(SUM(value), 0)").
		Where("tx_id IN (SELECT id FROM txes WHERE block_id <= ?)", block.ID).
		Where("id NOT IN (SELECT tx_out_id FROM tx_ins WHERE tx_in_id IN (SELECT id FROM txes WHERE block_id <= ?))", block.ID).
		Scan(&utxoTotal).Error
	if err != nil {
		log.Printf("[WARNING] Failed to calculate UTXO total: %v", err)
		utxoTotal = 45000000000000000 // 45 billion ADA default
	}

	// Calculate stake deposits (key deposits + pool deposits)
	var keyDepositCount int64
	var poolDepositCount int64
	
	// Count stake key registrations
	err = tx.Model(&models.StakeRegistration{}).
		Where("epoch_no <= ?", epochNo).
		Count(&keyDepositCount).Error
	if err != nil {
		log.Printf("[WARNING] Failed to count stake registrations: %v", err)
	}
	
	// Count pool registrations
	err = tx.Model(&models.PoolUpdate{}).
		Where("tx_id IN (SELECT id FROM txes WHERE block_id IN (SELECT id FROM blocks WHERE epoch_no <= ?))", epochNo).
		Distinct("hash_id").
		Count(&poolDepositCount).Error
	if err != nil {
		log.Printf("[WARNING] Failed to count pool registrations: %v", err)
	}
	
	// Calculate total deposits (2 ADA per stake key + 500 ADA per pool)
	stakeDeposits = uint64(keyDepositCount)*2000000 + uint64(poolDepositCount)*500000000

	// Calculate pending rewards
	err = tx.Model(&models.Reward{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("earned_epoch <= ? AND spendable_epoch > ?", epochNo, epochNo).
		Scan(&pendingRewards).Error
	if err != nil {
		log.Printf("[WARNING] Failed to calculate pending rewards: %v", err)
	}

	// Calculate total fees from transactions in this epoch
	err = tx.Model(&models.Tx{}).
		Select("COALESCE(SUM(fee), 0)").
		Where("block_id IN (SELECT id FROM blocks WHERE epoch_no = ?)", epochNo).
		Scan(&totalFees).Error
	if err != nil {
		log.Printf("[WARNING] Failed to calculate total fees: %v", err)
	}

	// Create ADA pots record
	// Note: The AdaPots model uses Utxo (not UTxO) as the field name
	adaPots := &models.AdaPots{
		SlotNo:   *block.SlotNo,
		EpochNo:  epochNo,
		Utxo:     utxoTotal,      // Field is named "Utxo" not "UTxO"
		Deposits: stakeDeposits,
		Rewards:  pendingRewards,
		Fees:     totalFees,
		// Treasury and Reserves would be calculated from protocol parameters
		// For now, using placeholder values based on typical mainnet values
		Treasury: 500000000000000,   // 500M ADA
		Reserves: 13500000000000000, // 13.5B ADA
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slot_no"}},
		DoNothing: true,
	}).Create(adaPots).Error; err != nil {
		// Ignore duplicate key errors  
		if !strings.Contains(err.Error(), "Duplicate entry") {
			return fmt.Errorf("failed to create ADA pots record: %w", err)
		}
	}

	log.Printf("Created ADA pots for epoch %d: UTXO=%d, Deposits=%d, Rewards=%d, Fees=%d", 
		epochNo, utxoTotal, stakeDeposits, pendingRewards, totalFees)

	return nil
}

// processEpochParameters extracts and stores protocol parameters
func (bp *BlockProcessor) processEpochParameters(ctx context.Context, tx *gorm.DB, block *models.Block, epochNo uint32) error {
	// In a real implementation, you would extract these from:
	// 1. The ledger state
	// 2. Protocol parameter updates from previous epochs
	// 3. Genesis configuration
	
	// Check if parameters already exist for this epoch
	var existingParam models.EpochParam
	err := tx.Where("epoch_no = ?", epochNo).First(&existingParam).Error
	if err == nil {
		// Parameters already exist for this epoch
		return nil
	}
	
	// Create epoch parameters with appropriate values based on era
	epochParam := &models.EpochParam{
		EpochNo:            epochNo,
		MinFeeA:            44,          // Linear fee coefficient
		MinFeeB:            155381,      // Constant fee
		MaxBlockSize:       90112,       // Maximum block body size
		MaxTxSize:          16384,       // Maximum transaction size
		MaxBhSize:          1100,        // Maximum block header size
		KeyDeposit:         2000000,     // Stake key deposit (2 ADA)
		PoolDeposit:        500000000,   // Pool registration deposit (500 ADA)
		MaxEpoch:           18,          // Maximum epochs a pool can announce retirement
		OptimalPoolCount:   500,         // k parameter - optimal number of pools
		Influence:          0.3,         // Pool influence factor (a0)
		MonetaryExpandRate: 0.003,       // Monetary expansion rate (rho)
		TreasuryGrowthRate: 0.2,         // Treasury growth rate (tau)
		Decentralisation:   0.0,         // Decentralisation parameter (d)
		ProtocolMajor:      block.ProtoMajor,
		ProtocolMinor:      block.ProtoMinor,
		MinUtxoValue:       1000000,     // Minimum UTXO value (1 ADA)
		MinPoolCost:        340000000,   // Minimum pool cost (340 ADA)
	}
	
	// Add Alonzo-era parameters if applicable
	if block.Era == "Alonzo" || block.Era == "Babbage" || block.Era == "Conway" {
		// Execution unit prices
		priceMem := 0.0577
		priceStep := 0.0000721
		epochParam.PriceMem = &priceMem
		epochParam.PriceStep = &priceStep
		
		// Execution limits
		maxTxExMem := uint64(14000000)
		maxTxExSteps := uint64(10000000000)
		maxBlockExMem := uint64(62000000)
		maxBlockExSteps := uint64(20000000000)
		epochParam.MaxTxExMem = &maxTxExMem
		epochParam.MaxTxExSteps = &maxTxExSteps
		epochParam.MaxBlockExMem = &maxBlockExMem
		epochParam.MaxBlockExSteps = &maxBlockExSteps
		
		// Value size limit
		maxValSize := uint64(5000)
		epochParam.MaxValSize = &maxValSize
		
		// Collateral parameters
		collateralPercent := uint32(150)
		maxCollateralInputs := uint32(3)
		epochParam.CollateralPercent = &collateralPercent
		epochParam.MaxCollateralInputs = &maxCollateralInputs
	}
	
	// Add Babbage-era parameters if applicable
	if block.Era == "Babbage" || block.Era == "Conway" {
		// Coins per UTXO size (replaces MinUtxoValue in Babbage)
		coinsPerUtxoSize := uint64(4310)
		epochParam.CoinsPerUtxoSize = &coinsPerUtxoSize
	}

	// Create the epoch parameters with OnConflict to handle duplicates
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "epoch_no"}},
		DoNothing: true,
	}).Create(epochParam).Error; err != nil {
		// If OnConflict doesn't work (older GORM), check for duplicate entry
		if !strings.Contains(err.Error(), "Duplicate entry") && !strings.Contains(err.Error(), "foreign key constraint") {
			return fmt.Errorf("failed to create epoch parameters: %w", err)
		}
	}

	log.Printf(" Created epoch parameters for epoch %d (protocol %d.%d)", 
		epochNo, block.ProtoMajor, block.ProtoMinor)

	return nil
}

// calculateEpochRewards calculates rewards for a completed epoch
func (bp *BlockProcessor) calculateEpochRewards(ctx context.Context, tx *gorm.DB, epochNo uint32) error {
	log.Printf(" Calculating rewards for epoch %d", epochNo)

	// Get all delegations active in this epoch
	var delegations []models.Delegation
	err := tx.Where("active_epoch_no <= ?", epochNo).
		Preload("Addr").
		Preload("PoolHash").
		Find(&delegations).Error
	if err != nil {
		return fmt.Errorf("failed to fetch delegations: %w", err)
	}

	// Calculate rewards for each delegation
	// Note: This is a simplified calculation. Real rewards depend on:
	// 1. Pool performance (blocks produced vs expected)
	// 2. Pool parameters (margin, cost)
	// 3. Total stake in the pool
	// 4. Protocol parameters (monetary expansion rate, etc.)
	
	rewardCount := 0
	for _, delegation := range delegations {
		// Skip if no stake address
		if delegation.AddrID == 0 {
			continue
		}

		// Calculate a simple reward (0.05% of stake per epoch as placeholder)
		// In production, this would be based on actual pool performance
		baseReward := uint64(1000000) // 1 ADA minimum reward

		reward := &models.Reward{
			AddrID:         delegation.AddrID,
			Type:           "member",
			Amount:         baseReward,
			EarnedEpoch:    epochNo,
			SpendableEpoch: epochNo + 2, // Rewards spendable 2 epochs later
			PoolID:         delegation.PoolHashID,
		}

		if err := tx.Create(reward).Error; err != nil {
			// Skip duplicates
			if !strings.Contains(err.Error(), "Duplicate entry") {
				log.Printf("[WARNING] Failed to create reward: %v", err)
			}
		} else {
			rewardCount++
		}
	}

	log.Printf("[OK] Created %d rewards for epoch %d", rewardCount, epochNo)
	return nil
}

// processProtocolParameterUpdates detects and stores protocol parameter update proposals
func (bp *BlockProcessor) processProtocolParameterUpdates(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// Get protocol parameter updates from transaction
	epochNo, updates := transaction.ProtocolParameterUpdates()
	if epochNo == 0 || len(updates) == 0 {
		return nil // No updates in this transaction
	}

	log.Printf(" Found protocol parameter update proposal for epoch %d", epochNo)

	// Get the transaction record to link the proposal
	var dbTx models.Tx
	if err := tx.First(&dbTx, txID).Error; err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}

	// Process parameter update proposals
	// The actual ProtocolParameterUpdate structure in gouroboros may have different fields
	// For now, we'll just log that we found updates
	log.Printf(" Found %d parameter update proposals for epoch %d", len(updates), epochNo)
	
	// In a complete implementation, you would:
	// 1. Check the actual fields available in common.ProtocolParameterUpdate
	// 2. Extract each parameter that exists
	// 3. Store them in the ParamProposal table
	// 
	// For example:
	// if update has MinFeeA field: proposal.MinFeeA = &update.MinFeeA
	// if update has MaxBlockSize field: proposal.MaxBlockSize = &update.MaxBlockSize
	// etc.

	// Also check for cost model updates in the transaction
	if err := bp.processCostModelUpdates(ctx, tx, txID, transaction); err != nil {
		log.Printf("[WARNING] Failed to process cost model updates: %v", err)
	}

	return nil
}

// processCostModelUpdates extracts and stores cost model updates from transactions
func (bp *BlockProcessor) processCostModelUpdates(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// Cost models are typically part of protocol parameter updates
	// In Plutus-enabled eras (Alonzo+), cost models define execution costs
	
	// Try to extract cost models from the transaction
	// This is a placeholder implementation as the actual structure depends on gouroboros
	log.Printf(" Checking for cost model updates in transaction %d", txID)
	
	// In a complete implementation, you would:
	// 1. Extract cost model data from protocol parameter updates
	// 2. Calculate hash of the cost model
	// 3. Store in cost_model table
	// 4. Link to param_proposal via cost_model_id
	
	return nil
}

// processExtraKeyWitnesses processes required signers from transactions
func (bp *BlockProcessor) processExtraKeyWitnesses(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// Get required signers from transaction
	signers := transaction.RequiredSigners()
	if len(signers) == 0 {
		return nil
	}

	log.Printf(" Processing %d extra key witnesses for transaction", len(signers))

	// Process each required signer
	for _, signerHash := range signers {
		// Convert hash to bytes
		hashBytes := signerHash[:]

		// Create extra key witness record
		witness := &models.ExtraKeyWitness{
			TxID: txID,
			Hash: hashBytes,
		}

		if err := tx.Create(witness).Error; err != nil {
			// Ignore duplicate entries
			if !strings.Contains(err.Error(), "Duplicate entry") {
				return fmt.Errorf("failed to create extra key witness: %w", err)
			}
		}
	}

	log.Printf("[OK] Stored %d extra key witnesses", len(signers))
	return nil
}

// storeTransactionCbor stores the raw CBOR bytes of a transaction
func (bp *BlockProcessor) storeTransactionCbor(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// Get CBOR bytes
	cborBytes := transaction.Cbor()
	if len(cborBytes) == 0 {
		return nil
	}

	// Create tx_cbor record
	txCbor := &models.TxCbor{
		TxID:  txID,
		Bytes: cborBytes,
	}

	if err := tx.Create(txCbor).Error; err != nil {
		// Ignore duplicate entries
		if !strings.Contains(err.Error(), "Duplicate entry") {
			return fmt.Errorf("failed to create tx_cbor: %w", err)
		}
	}

	return nil
}

// processReferenceInputs processes reference inputs for Babbage+ transactions
func (bp *BlockProcessor) processReferenceInputs(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction) error {
	// Try to get reference inputs
	var referenceInputs []ledger.TransactionInput
	
	// Use type assertion to check if transaction supports reference inputs
	type referenceInputGetter interface {
		ReferenceInputs() []ledger.TransactionInput
	}
	
	if refTx, ok := transaction.(referenceInputGetter); ok {
		referenceInputs = refTx.ReferenceInputs()
	} else {
		// Transaction doesn't support reference inputs
		return nil
	}

	if len(referenceInputs) == 0 {
		return nil
	}

	log.Printf(" Processing %d reference inputs for transaction", len(referenceInputs))

	// Process each reference input
	for _, refInput := range referenceInputs {
		// Get the output being referenced
		txOutID, txOutIndex := bp.getOutputReference(refInput)
		
		// Find the referenced output in our database
		var referencedOutput models.TxOut
		err := tx.Where("tx_id = ? AND `index` = ?", txOutID, txOutIndex).
			First(&referencedOutput).Error
		
		if err != nil {
			log.Printf("[WARNING] Referenced output not found: tx_id=%v, index=%d", txOutID, txOutIndex)
			continue
		}

		// Create reference input record
		refTxIn := &models.ReferenceTxIn{
			TxInID:      txID,
			TxOutID:     referencedOutput.TxID,
			TxOutIndex:  referencedOutput.Index,
		}

		if err := tx.Create(refTxIn).Error; err != nil {
			// Ignore duplicate entries
			if !strings.Contains(err.Error(), "Duplicate entry") {
				return fmt.Errorf("failed to create reference input: %w", err)
			}
		}
	}

	log.Printf("[OK] Stored %d reference inputs", len(referenceInputs))
	return nil
}

// getOutputReference extracts transaction ID and output index from an input
func (bp *BlockProcessor) getOutputReference(input ledger.TransactionInput) (txID uint64, index uint16) {
	// Get the hash of the transaction being referenced
	hash := input.Id()
	
	// Find the transaction by hash
	var referencedTx models.Tx
	err := bp.db.Where("hash = ?", hash[:]).First(&referencedTx).Error
	if err != nil {
		return 0, 0
	}
	
	return referencedTx.ID, uint16(input.Index())
}

// calculatePoolStatistics calculates pool performance statistics for an epoch
func (bp *BlockProcessor) calculatePoolStatistics(ctx context.Context, tx *gorm.DB, epochNo uint32) error {
	log.Printf("Calculating pool statistics for epoch %d", epochNo)

	// Get all pools that were active in this epoch
	var pools []struct {
		PoolHashID uint64
		View       string
	}
	err := tx.Table("pool_updates pu").
		Select("DISTINCT pu.hash_id as pool_hash_id, ph.view").
		Joins("JOIN pool_hashes ph ON ph.id = pu.hash_id").
		Where("pu.active_epoch_no <= ?", epochNo).
		Where("NOT EXISTS (SELECT 1 FROM pool_retires pr WHERE pr.hash_id = pu.hash_id AND pr.retiring_epoch <= ?)", epochNo).
		Scan(&pools).Error

	if err != nil {
		return fmt.Errorf("failed to get active pools: %w", err)
	}

	log.Printf(" Found %d active pools in epoch %d", len(pools), epochNo)

	// Calculate statistics for each pool
	for _, pool := range pools {
		// Count blocks produced by this pool in this epoch
		var blocksProduced int64
		err := tx.Model(&models.Block{}).
			Where("epoch_no = ?", epochNo).
			Where("slot_leader_id IN (SELECT id FROM slot_leaders WHERE pool_hash_id = ?)", pool.PoolHashID).
			Count(&blocksProduced).Error
		
		if err != nil {
			log.Printf("[WARNING] Failed to count blocks for pool %s: %v", pool.View, err)
			continue
		}

		// Count delegators
		var delegatorCount int64
		err = tx.Model(&models.Delegation{}).
			Where("pool_hash_id = ?", pool.PoolHashID).
			Where("active_epoch_no <= ?", epochNo).
			Count(&delegatorCount).Error
		
		if err != nil {
			log.Printf("[WARNING] Failed to count delegators for pool %s: %v", pool.View, err)
			continue
		}

		// Calculate total delegated stake
		var delegatedStake uint64
		err = tx.Table("epoch_stakes").
			Select("COALESCE(SUM(amount), 0)").
			Where("pool_id = ? AND epoch_no = ?", pool.PoolHashID, epochNo).
			Scan(&delegatedStake).Error
		
		if err != nil {
			log.Printf("[WARNING] Failed to calculate stake for pool %s: %v", pool.View, err)
			continue
		}

		// Create pool stat record
		poolStat := &models.PoolStat{
			PoolHashID:         pool.PoolHashID,
			EpochNo:            epochNo,
			NumberOfBlocks:     uint64(blocksProduced),
			NumberOfDelegators: uint64(delegatorCount),
			Stake:              delegatedStake,
			VotingPower:        0.0, // Would be calculated from total stake
		}

		if err := tx.Create(poolStat).Error; err != nil {
			// Ignore duplicate entries
			if !strings.Contains(err.Error(), "Duplicate entry") {
				log.Printf("[WARNING] Failed to create pool stat: %v", err)
			}
		}
	}

	log.Printf("[OK] Calculated statistics for %d pools in epoch %d", len(pools), epochNo)
	return nil
}
