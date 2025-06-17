package processors

import (
	"context"
	"fmt"
	"log"
	unifiederrors "nectar/errors"
	"nectar/models"
	"sync"
	"time"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Block type constants - These match the ledger era tags from gouroboros
const (
	BlockTypeByronEBB  = 0
	BlockTypeByronMain = 1
	BlockTypeShelley   = 2
	BlockTypeAllegra   = 3
	BlockTypeMary      = 4
	BlockTypeAlonzo    = 5
	BlockTypeBabbage   = 6
	BlockTypeConway    = 7
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
	txSemaphore          chan struct{} // Limits concurrent transaction processing
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
		txSemaphore:          make(chan struct{}, 32), // Increased to 32 for our powerful system
	}
	return bp
}

// ProcessBlock processes a single block and its transactions
func (bp *BlockProcessor) ProcessBlock(ctx context.Context, block ledger.Block, blockType uint) error {
	// Process block header in its own transaction
	var dbBlock *models.Block
	err := bp.db.Transaction(func(tx *gorm.DB) error {
		var err error
		dbBlock, err = bp.processBlockHeader(ctx, tx, block, blockType)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to process block header: %w", err)
	}

	// Process transactions with smaller scope (each tx in its own DB transaction)
	if err := bp.processEraAwareTransactions(ctx, bp.db, dbBlock, block, blockType); err != nil {
		return fmt.Errorf("failed to process transactions: %w", err)
	}

	return nil
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
		ProtoMajor:     0, // NOTE: Set by setEraSpecificBlockFields() based on era
		ProtoMinor:     0, // NOTE: Set by setEraSpecificBlockFields() based on era
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
func (bp *BlockProcessor) processEraAwareTransactions(ctx context.Context, db *gorm.DB, dbBlock *models.Block, block ledger.Block, blockType uint) error {
	transactions := block.Transactions()
	if len(transactions) == 0 {
		return nil
	}

	// Get block data for transactions
	blockHash := dbBlock.Hash
	blockNumber := block.BlockNumber()
	blockTime := dbBlock.Time

	// Process transactions concurrently with semaphore for concurrency control
	var wg sync.WaitGroup
	errChan := make(chan error, len(transactions))

	for idx, tx := range transactions {
		wg.Add(1)
		
		// Acquire semaphore
		bp.txSemaphore <- struct{}{}
		
		go func(idx int, tx ledger.Transaction) {
			defer wg.Done()
			defer func() { <-bp.txSemaphore }() // Release semaphore
			
			// Each transaction in its own DB transaction
			err := db.Transaction(func(dbTx *gorm.DB) error {
				return bp.processTransaction(ctx, dbTx, blockHash, blockNumber, blockTime, idx, tx, blockType)
			})
			if err != nil {
				errChan <- fmt.Errorf("transaction %d: %w", idx, err)
			}
		}(idx, tx)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err // Return first error
		}
	}

	return nil
}

// processTransaction processes a single transaction in its own DB transaction
func (bp *BlockProcessor) processTransaction(ctx context.Context, tx *gorm.DB, blockHash []byte, blockNumber uint64, blockTime time.Time, blockIndex int, transaction ledger.Transaction, blockType uint) error {
	hash := transaction.Hash()
	txHash := hash[:]

	// Create transaction record
	dbTx := models.Tx{
		Hash:       txHash,
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

	// Insert transaction
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}},
		DoNothing: true,
	}).Create(&dbTx).Error; err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Process transaction components in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, 6)

	// Process inputs (independent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bp.processTransactionInputs(ctx, tx, txHash, transaction, blockType); err != nil {
			errChan <- fmt.Errorf("inputs: %w", err)
		}
	}()

	// Process outputs (independent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bp.processTransactionOutputs(ctx, tx, txHash, transaction, blockType); err != nil {
			errChan <- fmt.Errorf("outputs: %w", err)
		}
	}()

	// Process metadata (independent)
	if metadata := transaction.Metadata(); metadata != nil {
		if metaValue := metadata.Value(); metaValue != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := bp.metadataProcessor.ProcessMetadata(tx, txHash, metadata); err != nil {
					errChan <- fmt.Errorf("metadata: %w", err)
				}
			}()
		}
	}

	// Process withdrawals (independent)
	if withdrawals := bp.getTransactionWithdrawals(transaction, blockType); len(withdrawals) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bp.withdrawalProcessor.ProcessWithdrawals(ctx, tx, txHash, withdrawals, blockType); err != nil {
				errChan <- fmt.Errorf("withdrawals: %w", err)
			}
		}()
	}

	// Process minting/burning (independent)
	if mint := bp.getTransactionMint(transaction, blockType); len(mint) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bp.assetProcessor.ProcessMint(tx, txHash, mint); err != nil {
				errChan <- fmt.Errorf("mint: %w", err)
			}
		}()
	}

	// Process scripts (independent)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bp.scriptProcessor.ProcessTransaction(tx, txHash, transaction, blockType); err != nil {
			errChan <- fmt.Errorf("scripts: %w", err)
		}
	}()

	// Process collateral inputs (independent, Alonzo+)
	if blockType >= BlockTypeAlonzo {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bp.processCollateralInputs(ctx, tx, txHash, transaction, blockType); err != nil {
				errChan <- fmt.Errorf("collateral inputs: %w", err)
			}
		}()
	}

	// Process collateral return (independent, Babbage+)
	if blockType >= BlockTypeBabbage {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bp.processCollateralReturn(ctx, tx, txHash, transaction, blockType); err != nil {
				errChan <- fmt.Errorf("collateral return: %w", err)
			}
		}()
	}

	// Process reference inputs (independent, Babbage+)
	if blockType >= BlockTypeBabbage {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bp.processReferenceInputs(ctx, tx, txHash, transaction, blockType); err != nil {
				errChan <- fmt.Errorf("reference inputs: %w", err)
			}
		}()
	}

	// Process required signers (independent, Alonzo+)
	if blockType >= BlockTypeAlonzo {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bp.processRequiredSigners(ctx, tx, txHash, transaction, blockType); err != nil {
				errChan <- fmt.Errorf("required signers: %w", err)
			}
		}()
	}

	// Wait for all independent components to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			// Log but don't fail the transaction
			unifiederrors.Get().Warning("BlockProcessor", "ProcessComponents", fmt.Sprintf("Component error in tx %d: %v", blockIndex, err))
		}
	}

	// Process dependent components (must be done after inputs/outputs)
	// Process certificates (may depend on stake addresses from outputs)
	if certs := bp.getTransactionCertificates(transaction, blockType); len(certs) > 0 {
		if err := bp.certificateProcessor.ProcessCertificates(ctx, tx, txHash, certs); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessCertificates", fmt.Sprintf("Failed to process certificates: %v", err))
		}
	}

	// Process governance actions (may depend on other components)
	if bp.hasGovernanceData(transaction, blockType) {
		if err := bp.governanceProcessor.ProcessTransaction(tx, txHash, transaction, blockType); err != nil {
			unifiederrors.Get().Warning("BlockProcessor", "ProcessGovernance", fmt.Sprintf("Failed to process governance: %v", err))
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
		
		// Safely get address string - handle potential panics from gouroboros
		var addressStr string
		func() {
			defer func() {
				if r := recover(); r != nil {
					// For problematic addresses, use base58 encoding of raw bytes as fallback
					if addrBytes, err := address.Bytes(); err == nil {
						addressStr = fmt.Sprintf("byron_raw_%x", addrBytes)
						// TODO: Consider rate limiting this warning in production
						log.Printf("[WARNING] Address.String() panicked, using raw format: %v", r)
					} else {
						addressStr = fmt.Sprintf("invalid_address_%d", i)
						log.Printf("[ERROR] Failed to get address string and bytes: %v", r)
					}
				}
			}()
			addressStr = address.String()
		}()
		
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

		// Extract stake address hash from base addresses
		if bp.extractStakeAddress(address) {
			// Get address bytes
			addrBytes, err := address.Bytes()
			if err == nil && len(addrBytes) >= 57 {
				// For base addresses, bytes 29-56 contain the stake credential hash
				stakeCredHash := addrBytes[29:57]
				dbOutput.StakeAddressHash = stakeCredHash
				
				// Also ensure the stake address is registered in the stake_addresses table
				// Create a reward address for this stake credential
				_, err := bp.stakeAddressCache.GetOrCreateStakeAddressFromBytesWithTx(tx, stakeCredHash)
				if err != nil {
					log.Printf("[WARNING] Failed to register stake address: %v", err)
				}
			}
		}

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

	// NOTE: Pool hash lookup from issuer vkey would require tracking
	// all stake pool registrations and their cold keys. For now,
	// we use the issuer vkey hash directly as the slot leader identifier.
	// This is sufficient for block attribution but doesn't link to
	// the actual stake pool entity.

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
	case BlockTypeShelley, BlockTypeAllegra, BlockTypeMary, BlockTypeAlonzo, BlockTypeBabbage, BlockTypeConway:
		return uint32(slot / 432000) // Shelley+ epoch = 432000 slots
	default:
		return uint32(slot / 432000) // Default to Shelley+ epoch length
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
	case BlockTypeShelley, BlockTypeAllegra, BlockTypeMary, BlockTypeAlonzo, BlockTypeBabbage, BlockTypeConway:
		// Shelley+ era: 1 second slots
		// Adjust for Byron slots (4492800 Byron slots at 20s each)
		shelleySlot := slot - 4492800
		return shelleyStart.Add(time.Duration(shelleySlot) * time.Second)
	default:
		// Default to Shelley+ era timing
		shelleySlot := slot - 4492800
		return shelleyStart.Add(time.Duration(shelleySlot) * time.Second)
	}
}

func (bp *BlockProcessor) getEraName(blockType uint) string {
	// The blockType parameter from gouroboros represents the ledger era tag
	// This mapping has been verified and corrected for all eras.
	
	// DEBUG: Uncomment to verify blockType values during development:
	// log.Printf("[DEBUG] getEraName called with blockType: %d", blockType)
	
	switch blockType {
	case BlockTypeByronEBB, BlockTypeByronMain:
		return "Byron"
	case BlockTypeShelley:
		return "Shelley"
	case BlockTypeAllegra: // FIXED: Allegra is blockType 3, not 4
		return "Allegra"
	case BlockTypeMary:
		return "Mary"
	case BlockTypeAlonzo:
		return "Alonzo"
	case BlockTypeBabbage:
		return "Babbage"
	case BlockTypeConway:
		return "Conway"
	default:
		// Log unknown blockType for debugging
		log.Printf("[WARNING] Unknown blockType: %d", blockType)
		return "Unknown"
	}
}

func (bp *BlockProcessor) isEpochBoundary(slot uint64, blockType uint) bool {
	switch blockType {
	case BlockTypeByronEBB:
		return true // EBB blocks are by definition epoch boundaries
	case BlockTypeByronMain:
		return slot%21600 == 0
	case BlockTypeShelley, BlockTypeAllegra, BlockTypeMary, BlockTypeAlonzo, BlockTypeBabbage, BlockTypeConway:
		return slot%432000 == 0
	default:
		return slot%432000 == 0
	}
}

// Stub functions for era-specific processing
func (bp *BlockProcessor) setEraSpecificBlockFields(block *models.Block, ledgerBlock ledger.Block, blockType uint) {
	header := ledgerBlock.Header()
	
	// Set protocol version based on era
	// IMPORTANT: These are approximate protocol versions based on era boundaries.
	// Actual protocol versions are determined by on-chain governance votes and
	// protocol parameter updates which can happen mid-era. For exact protocol
	// versions, one would need to track protocol parameter update proposals
	// and their activation points. This approximation is sufficient for most
	// indexing purposes but should not be used for protocol-critical decisions.
	switch blockType {
	case BlockTypeByronEBB, BlockTypeByronMain:
		// Byron uses protocol version 0.0 or 1.0
		block.ProtoMajor = 1
		block.ProtoMinor = 0
	case BlockTypeShelley:
		// Shelley: protocol version 2.0
		block.ProtoMajor = 2
		block.ProtoMinor = 0
	case BlockTypeAllegra:
		// Allegra: protocol version 3.0
		block.ProtoMajor = 3
		block.ProtoMinor = 0
	case BlockTypeMary:
		// Mary: protocol version 4.0
		block.ProtoMajor = 4
		block.ProtoMinor = 0
	case BlockTypeAlonzo:
		// Alonzo: protocol version 5.0 or 6.0
		block.ProtoMajor = 6
		block.ProtoMinor = 0
	case BlockTypeBabbage:
		// Babbage: protocol version 7.0 or 8.0
		block.ProtoMajor = 8
		block.ProtoMinor = 0
	case BlockTypeConway:
		// Conway: protocol version 9.0+
		block.ProtoMajor = 9
		block.ProtoMinor = 0
	}
	
	// Extract VRF key for Shelley+ blocks
	if blockType >= BlockTypeShelley {
		switch h := header.(type) {
		case *ledger.ShelleyBlockHeader:
			if h.Body.VrfKey != nil {
				block.VrfKey = stringPtr(fmt.Sprintf("%x", h.Body.VrfKey))
			}
		case *ledger.BabbageBlockHeader:
			if h.Body.VrfKey != nil {
				block.VrfKey = stringPtr(fmt.Sprintf("%x", h.Body.VrfKey))
			}
		}
	}
}

func (bp *BlockProcessor) setEraSpecificTxFields(tx *models.Tx, ledgerTx ledger.Transaction, blockType uint) {
	// Era-specific transaction fields
	switch blockType {
	case BlockTypeAllegra, BlockTypeMary, BlockTypeAlonzo, BlockTypeBabbage, BlockTypeConway:
		// Validity interval (introduced in Allegra)
		validity := ledgerTx.ValidityIntervalStart()
		if validity > 0 {
			validityInt := int64(validity)
			tx.InvalidBefore = &validityInt
		}
		
		// TTL (InvalidHereafter)
		if ttl := ledgerTx.TTL(); ttl > 0 {
			ttlInt := int64(ttl)
			tx.InvalidHereafter = &ttlInt
		}
	}
	
	// Alonzo+ specific fields
	if blockType >= BlockTypeAlonzo {
		// Script validity
		if ledgerTx.IsValid() {
			valid := true
			tx.ValidContract = &valid
		} else {
			valid := false
			tx.ValidContract = &valid
		}
		
		// Note: Collateral inputs are handled separately in the transaction processing
		// Reference inputs would need to be tracked in a separate table
		
		// Total collateral amount
		if totalCollateral := ledgerTx.TotalCollateral(); totalCollateral > 0 {
			totalCollateralInt := int64(totalCollateral)
			tx.TotalCollateral = &totalCollateralInt
		}
	}
	
	// Set deposit amount if applicable (handled in base transaction processing)
	
	// Treasury donation for Conway era
	if blockType >= BlockTypeConway {
		if donation := ledgerTx.Donation(); donation > 0 {
			donationInt := int64(donation)
			tx.TreasuryDonation = &donationInt
		}
	}
}

func (bp *BlockProcessor) processEpochBoundary(ctx context.Context, tx *gorm.DB, block *models.Block, epochNo uint32, blockType uint) error {
	// Only process epoch boundaries for Shelley+ eras
	if blockType < BlockTypeShelley {
		return nil
	}
	
	// Create epoch record
	epoch := &models.Epoch{
		No:        epochNo,
		StartTime: block.Time,
		EndTime:   block.Time.Add(5 * 24 * time.Hour), // Approximate 5-day epoch
	}
	
	// Insert or update epoch
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "no"}},
		UpdateAll: true,
	}).Create(epoch).Error; err != nil {
		return fmt.Errorf("failed to create epoch: %w", err)
	}
	
	// Log epoch transition
	log.Printf("[INFO] Epoch boundary: entering epoch %d at slot %d", epochNo, *block.SlotNo)
	
	// Note: Full epoch boundary processing would include:
	// - Snapshot stake distribution
	// - Calculate rewards
	// - Update pool parameters
	// - Process protocol parameter updates
	// These require additional state query integration
	
	return nil
}

func (bp *BlockProcessor) getTransactionCertificates(tx ledger.Transaction, blockType uint) []interface{} {
	// Certificates are only available from Shelley onwards
	if blockType < BlockTypeShelley {
		return nil
	}
	
	// Get certificates from transaction
	certs := tx.Certificates()
	if len(certs) == 0 {
		return nil
	}
	
	// Return certificates as interface slice for the certificate processor
	result := make([]interface{}, len(certs))
	for i, cert := range certs {
		result[i] = cert
	}
	
	return result
}

func (bp *BlockProcessor) getTransactionWithdrawals(tx ledger.Transaction, blockType uint) map[string]uint64 {
	// Withdrawals are only available from Shelley onwards
	if blockType < BlockTypeShelley {
		return nil
	}
	
	// Get withdrawals from transaction
	withdrawals := tx.Withdrawals()
	if withdrawals == nil || len(withdrawals) == 0 {
		return nil
	}
	
	// Convert to string map for the withdrawal processor
	result := make(map[string]uint64)
	for addr, amount := range withdrawals {
		// Convert address to string (bech32 format)
		addrStr := addr.String()
		result[addrStr] = amount
	}
	
	return result
}

func (bp *BlockProcessor) getTransactionMint(tx ledger.Transaction, blockType uint) map[string]int64 {
	// Multi-assets minting is only available from Mary era onwards
	if blockType < BlockTypeMary {
		return nil
	}
	
	// Get mint data from transaction
	mintAssets := tx.AssetMint()
	if mintAssets == nil {
		return nil
	}
	
	// Convert to our expected format
	result := make(map[string]int64)
	
	// Iterate through all policies
	for _, policyID := range mintAssets.Policies() {
		policyIDHex := fmt.Sprintf("%x", policyID[:])
		
		// Get all assets for this policy
		assetNames := mintAssets.Assets(policyID)
		for _, assetName := range assetNames {
			assetNameHex := fmt.Sprintf("%x", assetName)
			
			// Get the mint amount (positive for minting, negative for burning)
			amount := mintAssets.Asset(policyID, assetName)
			
			// Create key in format "policyId.assetName"
			key := fmt.Sprintf("%s.%s", policyIDHex, assetNameHex)
			
			// Store the actual amount as-is (positive for mint, negative for burn)
			result[key] = amount
		}
	}
	
	if len(result) == 0 {
		return nil
	}
	
	return result
}

func (bp *BlockProcessor) hasGovernanceData(tx ledger.Transaction, blockType uint) bool {
	// Governance data is only available from Conway era onwards
	if blockType < BlockTypeConway {
		return false
	}
	
	// Check if transaction has voting procedures
	if tx.VotingProcedures() != nil {
		return true
	}
	
	// Check if transaction has proposal procedures
	if proposals := tx.ProposalProcedures(); len(proposals) > 0 {
		return true
	}
	
	// Check if transaction has Conway-specific certificates
	certs := tx.Certificates()
	for _, cert := range certs {
		certType := cert.Type()
		// Conway certificate types: 8-17
		if certType >= 8 && certType <= 17 {
			return true
		}
	}
	
	return false
}

func (bp *BlockProcessor) getInputRedeemer(tx ledger.Transaction, index int, blockType uint) interface{} {
	// Redeemers are only available from Alonzo era onwards
	if blockType < BlockTypeAlonzo {
		return nil
	}
	
	// Get witnesses which contain redeemers
	witnesses := tx.Witnesses()
	if witnesses == nil {
		return nil
	}
	
	// Get redeemers from witnesses
	// Note: The exact method to access redeemers depends on the transaction type
	// For now, return nil as the interface may vary between transaction versions
	
	// GOUROBOROS LIMITATION: Redeemer access not yet exposed
	// The transaction witnesses contain redeemers, but gouroboros
	// doesn't currently expose methods to access them directly.
	// When available, we need to:
	// - Get redeemers array from witnesses
	// - Find redeemer with Purpose=Spend and Index=index
	// - Return the redeemer object for storage
	
	return nil
}

func (bp *BlockProcessor) getOutputInlineDatum(output ledger.TransactionOutput, blockType uint) []byte {
	// Inline datums are only available from Babbage era onwards
	if blockType < BlockTypeBabbage {
		return nil
	}
	
	// Get inline datum from output
	datum := output.Datum()
	if datum == nil {
		return nil
	}
	
	// Calculate hash of the datum for storage
	datumBytes := datum.Cbor()
	if len(datumBytes) == 0 {
		return nil
	}
	
	// Return the hash of the datum
	hash := ledger.Blake2b256(datumBytes)
	return hash[:]
}

func (bp *BlockProcessor) getOutputReferenceScript(output ledger.TransactionOutput, blockType uint) []byte {
	// Reference scripts are only available from Babbage era onwards
	if blockType < BlockTypeBabbage {
		return nil
	}
	
	// Try to extract reference script using type assertion to Babbage output
	switch o := output.(type) {
	case interface{ ScriptRef() *cbor.Tag }:
		// This interface doesn't exist yet in gouroboros
		// We need to use reflection or wait for gouroboros update
		return nil
	default:
		// GOUROBOROS LIMITATION: ScriptRef field not exposed
		// The BabbageTransactionOutput struct contains a ScriptRef field
		// but it's not accessible through the TransactionOutput interface.
		// This prevents us from indexing reference scripts attached to outputs.
		// When gouroboros adds this to the interface, we can extract and
		// store the reference script hash.
		_ = o // avoid unused variable warning
		return nil
	}
}

func (bp *BlockProcessor) extractPaymentCredential(addr ledger.Address) []byte {
	// Extract payment credential hash from address bytes
	addrBytes, err := addr.Bytes()
	if err != nil || len(addrBytes) < 29 {
		return nil
	}
	
	// For most address types, bytes 1-28 contain the payment credential hash
	// (byte 0 is the address type/network byte)
	return addrBytes[1:29]
}

func (bp *BlockProcessor) extractStakeAddress(addr ledger.Address) bool {
	// For Shelley+ addresses, we need to check if this is a base address with stake component
	
	addrBytes, err := addr.Bytes()
	if err != nil || len(addrBytes) < 57 {
		// Not a base address with stake component
		return false
	}
	
	// Check address type (first byte)
	addrType := addrBytes[0] & 0xF0
	
	// Base addresses (type 0x00-0x07) contain stake credentials
	if addrType>>4 > 7 {
		return false
	}
	
	// This is a base address with stake component
	return true
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

// processCollateralInputs processes collateral inputs for a transaction
func (bp *BlockProcessor) processCollateralInputs(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, blockType uint) error {
	// Only Alonzo and later have collateral
	if blockType < BlockTypeAlonzo {
		return nil
	}
	
	collateral := transaction.Collateral()
	if len(collateral) == 0 {
		return nil
	}
	
	inputBatch := make([]models.CollateralTxIn, 0, len(collateral))
	
	for _, input := range collateral {
		inputHash := input.Id()
		dbInput := models.CollateralTxIn{
			TxInHash:  txHash,
			TxOutHash: inputHash[:],
			TxOutIndex: input.Index(),
		}
		inputBatch = append(inputBatch, dbInput)
	}
	
	if len(inputBatch) > 0 {
		if err := tx.CreateInBatches(inputBatch, 1000).Error; err != nil {
			return fmt.Errorf("failed to batch insert collateral inputs: %w", err)
		}
	}
	
	return nil
}

// processCollateralReturn processes the collateral return output for a transaction
func (bp *BlockProcessor) processCollateralReturn(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, blockType uint) error {
	// Only Babbage and later have collateral return
	if blockType < BlockTypeBabbage {
		return nil
	}
	
	collateralReturn := transaction.CollateralReturn()
	if collateralReturn == nil {
		return nil
	}
	
	address := collateralReturn.Address()
	
	// Safely get address string
	var addressStr string
	func() {
		defer func() {
			if r := recover(); r != nil {
				if addrBytes, err := address.Bytes(); err == nil {
					addressStr = fmt.Sprintf("byron_raw_%x", addrBytes)
					log.Printf("[WARNING] Collateral return address.String() panicked, using raw format: %v", r)
				} else {
					addressStr = "invalid_collateral_address"
					log.Printf("[ERROR] Failed to get collateral return address string and bytes: %v", r)
				}
			}
		}()
		addressStr = address.String()
	}()
	
	addressBytes, err := address.Bytes()
	if err != nil {
		addressBytes = []byte{}
	}
	
	dbOutput := models.CollateralTxOut{
		TxHash:           txHash,
		Index:            0, // Collateral return always has index 0
		Address:          addressStr,
		AddressRaw:       addressBytes,
		AddressHasScript: false, // Will be implemented when address type detection is added
		Value:            collateralReturn.Amount(),
	}
	
	// Extract payment credential
	if paymentCred := bp.extractPaymentCredential(address); paymentCred != nil {
		dbOutput.PaymentCred = paymentCred
	}
	
	// Extract stake address hash
	if bp.extractStakeAddress(address) {
		addrBytes, err := address.Bytes()
		if err == nil && len(addrBytes) >= 57 {
			stakeCredHash := addrBytes[29:57]
			dbOutput.StakeAddressHash = stakeCredHash
			
			// Register stake address
			_, err := bp.stakeAddressCache.GetOrCreateStakeAddressFromBytesWithTx(tx, stakeCredHash)
			if err != nil {
				log.Printf("[WARNING] Failed to register stake address from collateral return: %v", err)
			}
		}
	}
	
	// Handle datum hash if present
	if datumHash := collateralReturn.DatumHash(); datumHash != nil {
		dbOutput.DataHash = datumHash.Bytes()
	}
	
	// Handle inline datum (Babbage+)
	if inlineDatum := bp.getOutputInlineDatum(collateralReturn, blockType); inlineDatum != nil {
		dbOutput.InlineDatumHash = inlineDatum
	}
	
	// Handle reference script (Babbage+)
	if refScript := bp.getOutputReferenceScript(collateralReturn, blockType); refScript != nil {
		dbOutput.ReferenceScriptHash = refScript
	}
	
	// Process multi-assets if present
	if assets := collateralReturn.Assets(); assets != nil {
		// Store the collateral output first to get its ID
		if err := tx.Create(&dbOutput).Error; err != nil {
			return fmt.Errorf("failed to insert collateral return output: %w", err)
		}
		
		// Process assets with the output ID
		if err := bp.assetProcessor.ProcessCollateralOutputAssets(tx, txHash, 0, assets); err != nil {
			return fmt.Errorf("failed to process collateral return assets: %w", err)
		}
		
		// Return early since we already inserted the output
		return nil
	}
	
	// Insert the collateral return output
	if err := tx.Create(&dbOutput).Error; err != nil {
		return fmt.Errorf("failed to insert collateral return output: %w", err)
	}
	
	return nil
}

// processReferenceInputs processes reference inputs for a transaction
func (bp *BlockProcessor) processReferenceInputs(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, blockType uint) error {
	// Only Babbage and later have reference inputs
	if blockType < BlockTypeBabbage {
		return nil
	}
	
	referenceInputs := transaction.ReferenceInputs()
	if len(referenceInputs) == 0 {
		return nil
	}
	
	inputBatch := make([]models.ReferenceTxIn, 0, len(referenceInputs))
	
	for _, input := range referenceInputs {
		inputHash := input.Id()
		dbInput := models.ReferenceTxIn{
			TxInHash:  txHash,
			TxOutHash: inputHash[:],
			TxOutIndex: input.Index(),
		}
		inputBatch = append(inputBatch, dbInput)
	}
	
	if len(inputBatch) > 0 {
		if err := tx.CreateInBatches(inputBatch, 1000).Error; err != nil {
			return fmt.Errorf("failed to batch insert reference inputs: %w", err)
		}
	}
	
	return nil
}

// processRequiredSigners extracts and stores required signers for a transaction
func (bp *BlockProcessor) processRequiredSigners(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, blockType uint) error {
	// Only Alonzo and later have required signers
	if blockType < BlockTypeAlonzo {
		return nil
	}
	
	requiredSigners := transaction.RequiredSigners()
	if len(requiredSigners) == 0 {
		return nil
	}
	
	// Create batch of required signers
	signerBatch := make([]models.RequiredSigner, 0, len(requiredSigners))
	for i, signer := range requiredSigners {
		signerRecord := models.RequiredSigner{
			TxHash:      txHash,
			SignerIndex: uint32(i),
			SignerHash:  signer[:],
		}
		signerBatch = append(signerBatch, signerRecord)
	}
	
	// Batch insert required signers
	if len(signerBatch) > 0 {
		if err := tx.CreateInBatches(signerBatch, 100).Error; err != nil {
			return fmt.Errorf("failed to batch insert required signers: %w", err)
		}
	}
	
	return nil
}

// GetDB returns the database connection
func (bp *BlockProcessor) GetDB() *gorm.DB {
	return bp.db
}

