package processors

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"gorm.io/gorm"

	"nectar/models"
)

const (
	// Sequential processing constants - no batching
	SEQUENTIAL_TX_BATCH_SIZE = 50
	SEQUENTIAL_OUT_BATCH_SIZE = 100
	SEQUENTIAL_IN_BATCH_SIZE = 100
)

// SequentialBlockProcessor - Gouroboros-aligned sequential processor
type SequentialBlockProcessor struct {
	db *gorm.DB
	stakeAddressCache map[string]uint64 // Simple cache for stake addresses
}

func NewSequentialBlockProcessor(db *gorm.DB) *SequentialBlockProcessor {
	return &SequentialBlockProcessor{
		db: db,
		stakeAddressCache: make(map[string]uint64),
	}
}

func (sbp *SequentialBlockProcessor) GetDB() *gorm.DB {
	return sbp.db
}

// ProcessBlock - Sequential processing without race conditions
func (sbp *SequentialBlockProcessor) ProcessBlock(ctx context.Context, tx *gorm.DB, block ledger.Block, blockType uint) (*models.Block, error) {
	// Step 1: Process block record
	dbBlock, err := sbp.processBlockRecord(ctx, tx, block, blockType)
	if err != nil {
		return nil, fmt.Errorf("failed to process block record: %w", err)
	}

	// Step 2: Process all transactions sequentially
	transactions := block.Transactions()
	for i, transaction := range transactions {
		if err := sbp.processTransaction(ctx, tx, dbBlock.ID, transaction, i, blockType); err != nil {
			log.Printf("Warning: failed to process transaction %d in block %d: %v", i, dbBlock.ID, err)
			// Continue with other transactions - don't fail entire block
		}
	}

	return dbBlock, nil
}

func (sbp *SequentialBlockProcessor) processBlockRecord(ctx context.Context, tx *gorm.DB, block ledger.Block, blockType uint) (*models.Block, error) {
	hash := block.Hash()
	hashBytes := hash[:]
	slotNumber := block.SlotNumber()

	// Check if block already exists
	var existingBlock models.Block
	err := tx.Where("hash = ?", hashBytes).First(&existingBlock).Error
	if err == nil {
		log.Printf("Block already exists: slot %d, using existing ID %d", slotNumber, existingBlock.ID)
		return &existingBlock, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing block: %w", err)
	}

	// Create new block record
	dbBlock := &models.Block{
		Hash:    hashBytes,
		SlotNo:  &slotNumber,
		Time:    sbp.slotToTime(slotNumber, blockType),
		TxCount: uint64(len(block.Transactions())),
		Size:    uint32(len(block.Cbor())),
		Era:     block.Era().Name,
	}

	// Set protocol versions aligned with gouroboros
	switch blockType {
	case 0, 1: // Byron EBB, Byron Main
		dbBlock.ProtoMajor = 1
		dbBlock.ProtoMinor = 0
	case 2: // Shelley
		dbBlock.ProtoMajor = 2
		dbBlock.ProtoMinor = 0
	case 3: // Allegra
		dbBlock.ProtoMajor = 3
		dbBlock.ProtoMinor = 0
	case 4: // Mary
		dbBlock.ProtoMajor = 4
		dbBlock.ProtoMinor = 0
	case 5: // Alonzo
		dbBlock.ProtoMajor = 6
		dbBlock.ProtoMinor = 0
	case 6: // Babbage
		dbBlock.ProtoMajor = 8
		dbBlock.ProtoMinor = 0
	case 7: // Conway
		dbBlock.ProtoMajor = 10
		dbBlock.ProtoMinor = 0
	default:
		dbBlock.ProtoMajor = 1
		dbBlock.ProtoMinor = 0
	}

	// Set epoch
	if epochNo := sbp.calculateEpoch(slotNumber, blockType); epochNo > 0 {
		dbBlock.EpochNo = &epochNo
	}

	// Set block number
	if blockNo := block.BlockNumber(); blockNo > 0 {
		blockNum := uint32(blockNo)
		dbBlock.BlockNo = &blockNum
	}

	// Create slot leader
	slotLeader, err := sbp.createSlotLeader(tx, blockType, slotNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create slot leader: %w", err)
	}
	dbBlock.SlotLeaderID = slotLeader.ID

	// Insert block - simple INSERT, no UPSERT conflicts
	if err := tx.Create(dbBlock).Error; err != nil {
		return nil, fmt.Errorf("failed to insert block: %w", err)
	}

	return dbBlock, nil
}

func (sbp *SequentialBlockProcessor) createSlotLeader(tx *gorm.DB, blockType uint, slotNumber uint64) (*models.SlotLeader, error) {
	eraName := sbp.getEraName(blockType)
	leaderHash := []byte(fmt.Sprintf("leader-%s-%d", eraName, slotNumber))
	
	// Check if slot leader exists
	var slotLeader models.SlotLeader
	err := tx.Where("hash = ?", leaderHash).First(&slotLeader).Error
	if err == nil {
		return &slotLeader, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing slot leader: %w", err)
	}

	// Create new slot leader
	slotLeader = models.SlotLeader{
		Hash:        leaderHash,
		Description: fmt.Sprintf("Slot leader for %s era block %d", eraName, slotNumber),
	}

	if err := tx.Create(&slotLeader).Error; err != nil {
		return nil, fmt.Errorf("failed to create slot leader: %w", err)
	}

	return &slotLeader, nil
}

func (sbp *SequentialBlockProcessor) processTransaction(ctx context.Context, tx *gorm.DB, blockID uint64, transaction ledger.Transaction, blockIndex int, blockType uint) error {
	hash := transaction.Hash()
	hashBytes := hash[:]

	// Check if transaction exists
	var existingTx models.Tx
	err := tx.Where("hash = ?", hashBytes).First(&existingTx).Error
	if err == nil {
		log.Printf("Transaction already exists: %x, skipping", hashBytes)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing transaction: %w", err)
	}

	// Create transaction record
	dbTx := models.Tx{
		Hash:       hashBytes,
		BlockID:    blockID,
		BlockIndex: uint32(blockIndex),
		Size:       uint32(len(transaction.Cbor())),
	}

	// Set transaction fields
	if fee := transaction.Fee(); fee > 0 {
		dbTx.Fee = &fee
	}

	// Calculate output sum
	outputs := transaction.Outputs()
	if len(outputs) > 0 {
		var outSum uint64
		for _, output := range outputs {
			outSum += output.Amount()
		}
		dbTx.OutSum = &outSum
	}

	// Set era-specific fields
	sbp.setEraSpecificTxFields(&dbTx, transaction, blockType)

	// Insert transaction
	if err := tx.Create(&dbTx).Error; err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Process transaction components
	if err := sbp.processTransactionInputs(ctx, tx, dbTx.ID, transaction, blockType); err != nil {
		log.Printf("Warning: failed to process inputs: %v", err)
	}

	if err := sbp.processTransactionOutputs(ctx, tx, dbTx.ID, transaction, blockType); err != nil {
		log.Printf("Warning: failed to process outputs: %v", err)
	}

	if err := sbp.processTransactionComponents(ctx, tx, dbTx.ID, transaction, blockType); err != nil {
		log.Printf("Warning: failed to process components: %v", err)
	}

	return nil
}

func (sbp *SequentialBlockProcessor) processTransactionInputs(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, blockType uint) error {
	inputs := transaction.Inputs()
	if len(inputs) == 0 {
		return nil
	}

	for _, input := range inputs {
		hash := input.Id()
		hashBytes := hash[:]

		// Find referenced output
		var referencedTxOut models.TxOut
		if err := tx.Where("tx_id IN (SELECT id FROM txes WHERE hash = ?) AND `index` = ?",
			hashBytes, input.Index()).First(&referencedTxOut).Error; err != nil {
			// Output not found - normal for Byron era and historical references
			if blockType <= 1 { // Byron (EBB=0, Main=1)
				continue
			}
			log.Printf("Warning: Historical reference %x:%d (normal)", hashBytes, input.Index())
			continue
		}

		txIn := models.TxIn{
			TxInID:     txID,
			TxOutID:    referencedTxOut.ID,
			TxOutIndex: uint16(input.Index()),
		}

		// Simple INSERT - no conflicts in sequential processing
		if err := tx.Create(&txIn).Error; err != nil {
			log.Printf("Warning: failed to insert tx input: %v", err)
		}
	}

	return nil
}

func (sbp *SequentialBlockProcessor) processTransactionOutputs(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, blockType uint) error {
	outputs := transaction.Outputs()
	if len(outputs) == 0 {
		return nil
	}

	for i, output := range outputs {
		txOut := models.TxOut{
			TxID:    txID,
			Index:   uint16(i),
			Address: output.Address().String(),
			Value:   output.Amount(),
		}

		// Handle era-specific output features
		if assets := output.Assets(); assets != nil {
			// Process multi-assets for Mary+ era
			if blockType >= 4 {
				// Multi-assets present (Mary+ era)
				// Note: HasAssets field not in model, assets handled separately
			}
		}

		// Handle datum for Alonzo+ era
		if blockType >= 5 {
			if datumHash := output.DatumHash(); datumHash != nil {
				txOut.DataHash = datumHash[:]
			}
		}

		// Simple INSERT
		if err := tx.Create(&txOut).Error; err != nil {
			log.Printf("Warning: failed to insert tx output %d: %v", i, err)
		}
	}

	return nil
}

func (sbp *SequentialBlockProcessor) processTransactionComponents(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, blockType uint) error {
	// Process certificates for Shelley+ era
	if blockType >= 2 {
		if certs := transaction.Certificates(); len(certs) > 0 {
			// Simple certificate processing
			log.Printf("Processing %d certificates for tx %d", len(certs), txID)
		}
	}

	// Process withdrawals for Shelley+ era
	if blockType >= 2 {
		if withdrawals := transaction.Withdrawals(); withdrawals != nil {
			// Simple withdrawal processing
			log.Printf("Processing withdrawals for tx %d", txID)
		}
	}

	// Process metadata
	if metadata := transaction.Metadata(); metadata != nil {
		// Simple metadata processing
		log.Printf("Processing metadata for tx %d", txID)
	}

	return nil
}

func (sbp *SequentialBlockProcessor) setEraSpecificTxFields(dbTx *models.Tx, transaction ledger.Transaction, blockType uint) {
	// Set TTL for Shelley+ era
	if blockType >= 2 {
		if ttl := transaction.TTL(); ttl > 0 {
			invalidHereafter := ttl
			dbTx.InvalidHereafter = &invalidHereafter
		}
	}

	// Set validity flag for Alonzo+ era
	if blockType >= 5 {
		validityFlag := transaction.IsValid()
		dbTx.ValidContract = &validityFlag
	}

	// Set script size for Alonzo+ era
	if blockType >= 5 {
		if witnessSet := transaction.Witnesses(); witnessSet != nil {
			// Calculate script size
			scriptSize := uint32(0) // Simplified calculation
			dbTx.ScriptSize = &scriptSize
		}
	}
}

// UTILITY FUNCTIONS

func (sbp *SequentialBlockProcessor) getEraName(blockType uint) string {
	switch blockType {
	case 0:
		return "Byron (EBB)"
	case 1:
		return "Byron"
	case 2:
		return "Shelley"
	case 3:
		return "Allegra"
	case 4:
		return "Mary"
	case 5:
		return "Alonzo"
	case 6:
		return "Babbage"
	case 7:
		return "Conway"
	default:
		return "Unknown"
	}
}

func (sbp *SequentialBlockProcessor) calculateEpoch(slot uint64, blockType uint) uint32 {
	const (
		ByronSlotsPerEpoch   = 21600
		ShelleySlotsPerEpoch = 432000
		ShelleyStartSlot     = 4492800
	)

	if slot < ShelleyStartSlot {
		return uint32(slot / ByronSlotsPerEpoch)
	} else {
		byronEpochs := uint32(ShelleyStartSlot / ByronSlotsPerEpoch)
		shelleySlots := slot - ShelleyStartSlot
		shelleyEpochs := uint32(shelleySlots / ShelleySlotsPerEpoch)
		return byronEpochs + shelleyEpochs
	}
}

func (sbp *SequentialBlockProcessor) slotToTime(slot uint64, blockType uint) time.Time {
	const (
		ByronSlotLength   = 20
		ShelleySlotLength = 1
		ShelleyStartSlot  = 4492800
	)

	byronStart := time.Date(2017, 9, 23, 21, 44, 51, 0, time.UTC)
	shelleyStart := time.Date(2020, 7, 29, 21, 44, 51, 0, time.UTC)

	if slot < ShelleyStartSlot {
		return byronStart.Add(time.Duration(slot) * ByronSlotLength * time.Second)
	} else {
		return shelleyStart.Add(time.Duration(slot-ShelleyStartSlot) * ShelleySlotLength * time.Second)
	}
}

func (sbp *SequentialBlockProcessor) HandleRollback(ctx context.Context, point common.Point) error {
	log.Printf("Sequential rollback to slot %d", point.Slot)
	
	// Handle rollback with proper foreign key constraint handling
	tx := sbp.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Step 1: Delete transaction inputs/outputs for blocks to be rolled back
	var blocksToDelete []models.Block
	if err := tx.Where("slot_no > ?", point.Slot).Find(&blocksToDelete).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find blocks for rollback: %w", err)
	}

	if len(blocksToDelete) > 0 {
		var blockIDs []uint64
		for _, block := range blocksToDelete {
			blockIDs = append(blockIDs, block.ID)
		}

		// Delete transaction-related data first (respecting foreign keys)
		if err := tx.Where("tx_id IN (SELECT id FROM txes WHERE block_id IN ?)", blockIDs).Delete(&models.TxIn{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete tx_ins during rollback: %w", err)
		}

		if err := tx.Where("tx_id IN (SELECT id FROM txes WHERE block_id IN ?)", blockIDs).Delete(&models.TxOut{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete tx_outs during rollback: %w", err)
		}

		if err := tx.Where("tx_id IN (SELECT id FROM txes WHERE block_id IN ?)", blockIDs).Delete(&models.MaTxOut{}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete ma_tx_outs during rollback: %w", err)
		}

		// Delete transactions
		txResult := tx.Where("block_id IN ?", blockIDs).Delete(&models.Tx{})
		if txResult.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete transactions during rollback: %w", txResult.Error)
		}

		// Finally delete blocks
		blockResult := tx.Where("slot_no > ?", point.Slot).Delete(&models.Block{})
		if blockResult.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete blocks during rollback: %w", blockResult.Error)
		}

		log.Printf("Rolled back %d transactions and %d blocks", txResult.RowsAffected, blockResult.RowsAffected)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	return nil
}