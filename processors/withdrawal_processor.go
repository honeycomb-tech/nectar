package processors

import (
	"context"
	"fmt"
	"log"
	"nectar/models"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"gorm.io/gorm"
)

// WithdrawalProcessor handles processing reward withdrawals for all Cardano eras
type WithdrawalProcessor struct {
	db                *gorm.DB
	stakeAddressCache *StakeAddressCache
	errorCollector    *ErrorCollector
}

// NewWithdrawalProcessor creates a new withdrawal processor
func NewWithdrawalProcessor(db *gorm.DB) *WithdrawalProcessor {
	return &WithdrawalProcessor{
		db:                db,
		stakeAddressCache: NewStakeAddressCache(db),
		errorCollector:    GetGlobalErrorCollector(),
	}
}

// ProcessWithdrawals processes withdrawals from a transaction
func (wp *WithdrawalProcessor) ProcessWithdrawals(ctx context.Context, tx *gorm.DB, txID uint64, withdrawals map[string]uint64, blockType uint) error {
	if len(withdrawals) == 0 {
		return nil
	}

	if GlobalLoggingConfig.LogStakeOperations.Load() {
		log.Printf("Processing %d withdrawals for transaction %d", len(withdrawals), txID)
	}

	for rewardAccount, amount := range withdrawals {
		if err := wp.processWithdrawal(tx, txID, rewardAccount, amount); err != nil {
			log.Printf("[WARNING] Withdrawal processing error: %v", err)
			continue
		}
	}

	return nil
}

// processWithdrawal processes a single withdrawal
func (wp *WithdrawalProcessor) processWithdrawal(tx *gorm.DB, txID uint64, rewardAccountStr string, amount uint64) error {
	// Parse the reward account address
	rewardAddr, err := common.NewAddress(rewardAccountStr)
	if err != nil {
		return fmt.Errorf("failed to parse reward account %s: %w", rewardAccountStr, err)
	}

	// Extract stake key hash from the reward address
	stakeKeyHash := rewardAddr.StakeKeyHash()
	// Blake2b224 is an array type, so check if it's empty
	var emptyHash common.Blake2b224
	if stakeKeyHash == emptyHash {
		return fmt.Errorf("reward address %s does not contain stake key hash", rewardAccountStr)
	}

	// Get or create the stake address (using atomic FirstOrCreate with transaction context)
	stakeAddrID, err := wp.stakeAddressCache.GetOrCreateStakeAddressFromBytesWithTx(tx, stakeKeyHash[:])
	if err != nil {
		return fmt.Errorf("failed to get stake address: %w", err)
	}

	// Simple validation
	if stakeAddrID == 0 {
		return fmt.Errorf("invalid stake address ID: got 0")
	}

	// DEBUG: Verify stake address actually exists in database
	var existsCheck models.StakeAddress
	if err := tx.Where("id = ?", stakeAddrID).First(&existsCheck).Error; err != nil {
		log.Printf("[WARNING] STAKE ADDRESS VERIFICATION FAILED: ID %d does not exist in database (reward account: %s)", stakeAddrID, rewardAccountStr)
		log.Printf("[WARNING] Attempting to re-create stake address from hash: %x", stakeKeyHash[:])
		
		// Try to get/create again with the transaction context
		newStakeAddrID, createErr := wp.stakeAddressCache.GetOrCreateStakeAddressFromBytesWithTx(tx, stakeKeyHash[:])
		if createErr != nil {
			return fmt.Errorf("failed to re-create stake address: %w", createErr)
		}
		log.Printf("[OK] Re-created stake address with ID: %d", newStakeAddrID)
		stakeAddrID = newStakeAddrID
	} else {
		if GlobalLoggingConfig.LogStakeOperations.Load() {
			log.Printf("[OK] Verified stake address ID %d exists for withdrawal (reward account: %s)", stakeAddrID, rewardAccountStr[:16]+"...")
		}
	}

	// Create withdrawal record
	withdrawal := &models.Withdrawal{
		TxID:   txID,
		AddrID: stakeAddrID,
		Amount: amount,
	}

	if err := tx.Create(withdrawal).Error; err != nil {
		log.Printf("[WARNING] WITHDRAWAL CREATION FAILED: tx_id=%d, addr_id=%d, amount=%d, error=%v", txID, stakeAddrID, amount, err)
		return fmt.Errorf("failed to create withdrawal record: %w", err)
	}

	if GlobalLoggingConfig.LogStakeOperations.Load() {
		log.Printf("[OK] Processed withdrawal: %s -> %d lovelace", rewardAccountStr[:16]+"...", amount)
	}
	return nil
}

// ProcessWithdrawalsFromTransaction extracts and processes withdrawals from a transaction
func (wp *WithdrawalProcessor) ProcessWithdrawalsFromTransaction(ctx context.Context, tx *gorm.DB, txID uint64, transaction interface{}, blockType uint) error {
	// Only process withdrawals for Shelley+ eras
	if blockType < 2 {
		return nil
	}

	// Try to extract withdrawals using interface assertion
	switch txWithWithdrawals := transaction.(type) {
	case interface{ Withdrawals() map[*common.Address]uint64 }:
		withdrawalsMap := txWithWithdrawals.Withdrawals()
		// Convert map[*common.Address]uint64 to map[string]uint64
		withdrawals := make(map[string]uint64)
		for addr, amount := range withdrawalsMap {
			if addr != nil {
				withdrawals[addr.String()] = amount
			}
		}
		return wp.ProcessWithdrawals(ctx, tx, txID, withdrawals, blockType)
	default:
		// Transaction doesn't support withdrawals or has none
		return nil
	}
}

// BatchProcessWithdrawals processes multiple withdrawals in a batch
func (wp *WithdrawalProcessor) BatchProcessWithdrawals(ctx context.Context, tx *gorm.DB, withdrawalBatch []WithdrawalBatch) error {
	if len(withdrawalBatch) == 0 {
		return nil
	}

	log.Printf("Batch processing %d withdrawals", len(withdrawalBatch))

	// Pre-populate stake address cache for all unique addresses
	uniqueAddresses := make(map[string]bool)
	for _, item := range withdrawalBatch {
		uniqueAddresses[item.RewardAccount] = true
	}

	for addr := range uniqueAddresses {
		if err := wp.preloadStakeAddress(tx, addr); err != nil {
			log.Printf("[WARNING] Failed to preload stake address for %s: %v", addr, err)
		}
	}

	// Process withdrawals
	for _, item := range withdrawalBatch {
		if err := wp.processWithdrawal(tx, item.TxID, item.RewardAccount, item.Amount); err != nil {
			wp.errorCollector.ProcessingWarning("WithdrawalProcessor", "batchProcessWithdrawal",
				fmt.Sprintf("failed to process withdrawal: %v", err),
				fmt.Sprintf("tx_id:%d,reward_account:%s", item.TxID, item.RewardAccount))
			continue
		}
	}

	return nil
}

// WithdrawalBatch represents a batch of withdrawals to process
type WithdrawalBatch struct {
	TxID          uint64
	RewardAccount string
	Amount        uint64
}

// preloadStakeAddress preloads a stake address into the cache
func (wp *WithdrawalProcessor) preloadStakeAddress(tx *gorm.DB, rewardAccountStr string) error {
	rewardAddr, err := common.NewAddress(rewardAccountStr)
	if err != nil {
		return fmt.Errorf("failed to parse reward account: %w", err)
	}

	stakeKeyHash := rewardAddr.StakeKeyHash()
	_, err = wp.stakeAddressCache.GetOrCreateStakeAddressFromBytesWithTx(tx, stakeKeyHash[:])
	return err
}

// GetWithdrawalStats returns statistics about withdrawals
func (wp *WithdrawalProcessor) GetWithdrawalStats(tx *gorm.DB) (*WithdrawalStats, error) {
	var stats WithdrawalStats

	// Total withdrawals
	if err := tx.Model(&models.Withdrawal{}).Count(&stats.TotalWithdrawals).Error; err != nil {
		return nil, fmt.Errorf("failed to count withdrawals: %w", err)
	}

	// Total amount withdrawn
	if err := tx.Model(&models.Withdrawal{}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stats.TotalAmountWithdrawn).Error; err != nil {
		return nil, fmt.Errorf("failed to sum withdrawal amounts: %w", err)
	}

	// Unique addresses
	if err := tx.Model(&models.Withdrawal{}).
		Select("COUNT(DISTINCT addr_id)").
		Scan(&stats.UniqueAddresses).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique addresses: %w", err)
	}

	return &stats, nil
}

// WithdrawalStats contains withdrawal statistics
type WithdrawalStats struct {
	TotalWithdrawals     int64
	TotalAmountWithdrawn uint64
	UniqueAddresses      int64
}

// extractRewardAddressComponents extracts components from a reward address
func (wp *WithdrawalProcessor) extractRewardAddressComponents(rewardAccountStr string) (*RewardAddressComponents, error) {
	rewardAddr, err := common.NewAddress(rewardAccountStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reward address: %w", err)
	}

	stakeKeyHash := rewardAddr.StakeKeyHash()
	components := &RewardAddressComponents{
		Address:       rewardAccountStr,
		StakeKeyHash:  &stakeKeyHash,
		NetworkID:     uint8(0), // TODO: Extract network ID from address
		AddressType:   rewardAddr.Type(),
	}

	return components, nil
}

// RewardAddressComponents contains components of a reward address
type RewardAddressComponents struct {
	Address      string
	StakeKeyHash *common.Blake2b224
	NetworkID    uint8
	AddressType  uint8
}

// ValidateWithdrawal performs validation on a withdrawal
func (wp *WithdrawalProcessor) ValidateWithdrawal(rewardAccount string, amount uint64) error {
	// Validate reward account format
	if _, err := common.NewAddress(rewardAccount); err != nil {
		return fmt.Errorf("invalid reward account format: %w", err)
	}

	// Validate amount
	if amount == 0 {
		return fmt.Errorf("withdrawal amount cannot be zero")
	}

	// Maximum possible ADA supply is 45 billion
	const maxADASupply = 45_000_000_000_000_000 // in lovelace
	if amount > maxADASupply {
		return fmt.Errorf("withdrawal amount exceeds maximum possible: %d", amount)
	}

	return nil
}

// ProcessBulkWithdrawals processes withdrawals for multiple transactions efficiently
func (wp *WithdrawalProcessor) ProcessBulkWithdrawals(ctx context.Context, tx *gorm.DB, items []WithdrawalItem) error {
	if len(items) == 0 {
		return nil
	}

	// Group by transaction for logging
	txGroups := make(map[uint64][]WithdrawalItem)
	for _, item := range items {
		txGroups[item.TxID] = append(txGroups[item.TxID], item)
	}

	log.Printf("Processing withdrawals for %d transactions", len(txGroups))

	// Process each item
	for _, item := range items {
		rewardAddr, err := common.NewAddress(item.RewardAccount)
		if err != nil {
			wp.errorCollector.ProcessingWarning("WithdrawalProcessor", "processWithdrawal",
				fmt.Sprintf("failed to parse reward account: %v", err),
				fmt.Sprintf("tx_id:%d,reward_account:%s", item.TxID, item.RewardAccount))
			continue
		}

		stakeKeyHash := rewardAddr.StakeKeyHash()
		var emptyHash common.Blake2b224
		if stakeKeyHash == emptyHash {
			wp.errorCollector.ProcessingWarning("WithdrawalProcessor", "processWithdrawal",
				"reward address does not contain stake key hash",
				fmt.Sprintf("tx_id:%d,reward_account:%s", item.TxID, item.RewardAccount))
			continue
		}

		stakeAddrID, err := wp.stakeAddressCache.GetOrCreateStakeAddressFromBytesWithTx(tx, stakeKeyHash[:])
		if err != nil {
			wp.errorCollector.ProcessingWarning("WithdrawalProcessor", "processWithdrawal",
				fmt.Sprintf("failed to get stake address: %v", err),
				fmt.Sprintf("tx_id:%d,reward_account:%s", item.TxID, item.RewardAccount))
			continue
		}

		withdrawal := &models.Withdrawal{
			TxID:   item.TxID,
			AddrID: stakeAddrID,
			Amount: item.Amount,
		}

		if err := tx.Create(withdrawal).Error; err != nil {
			wp.errorCollector.ProcessingWarning("WithdrawalProcessor", "processWithdrawal",
				fmt.Sprintf("failed to create withdrawal record: %v", err),
				fmt.Sprintf("tx_id:%d,reward_account:%s", item.TxID, item.RewardAccount))
			continue
		}
	}

	return nil
}

// WithdrawalItem represents a withdrawal to be processed
type WithdrawalItem struct {
	TxID          uint64
	RewardAccount string
	Amount        uint64
}