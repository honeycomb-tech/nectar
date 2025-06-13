package processors

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"nectar/models"
	"sync"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"gorm.io/gorm"
)

// AssetProcessor handles processing multi-assets for Mary era and beyond
type AssetProcessor struct {
	db              *gorm.DB
	multiAssetCache *MultiAssetCache
	errorCollector  *ErrorCollector
}

// MultiAssetCache manages multi-asset lookups
type MultiAssetCache struct {
	cache map[string]uint64
	mutex sync.RWMutex
	db    *gorm.DB
}

// NewMultiAssetCache creates a new multi-asset cache
func NewMultiAssetCache(db *gorm.DB) *MultiAssetCache {
	return &MultiAssetCache{
		cache: make(map[string]uint64),
		db:    db,
	}
}

// GetOrCreateMultiAsset retrieves or creates a multi-asset ID
func (mac *MultiAssetCache) GetOrCreateMultiAsset(policyID []byte, assetName []byte) (uint64, error) {
	// Use the transaction context if available
	return mac.GetOrCreateMultiAssetWithTx(mac.db, policyID, assetName)
}

// GetOrCreateMultiAssetWithTx retrieves or creates a multi-asset ID with transaction context
func (mac *MultiAssetCache) GetOrCreateMultiAssetWithTx(tx *gorm.DB, policyID []byte, assetName []byte) (uint64, error) {
	cacheKey := fmt.Sprintf("%s:%s", hex.EncodeToString(policyID), hex.EncodeToString(assetName))
	
	// Check cache first
	mac.mutex.RLock()
	if id, exists := mac.cache[cacheKey]; exists {
		mac.mutex.RUnlock()
		return id, nil
	}
	mac.mutex.RUnlock()

	// Check database using transaction context
	var multiAsset models.MultiAsset
	result := tx.Where("policy = ? AND name = ?", policyID, assetName).First(&multiAsset)
	if result.Error == nil {
		// Found in database, cache it
		mac.mutex.Lock()
		mac.cache[cacheKey] = multiAsset.ID
		mac.mutex.Unlock()
		return multiAsset.ID, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("database error: %w", result.Error)
	}

	// Create new multi-asset using FirstOrCreate for atomicity
	multiAsset = models.MultiAsset{
		Policy:      policyID,
		Name:        assetName,
		Fingerprint: common.NewAssetFingerprint(policyID, assetName).String(),
	}

	// Use FirstOrCreate to handle race conditions properly
	if err := tx.Where("policy = ? AND name = ?", policyID, assetName).
		Attrs(models.MultiAsset{
			Policy:      policyID,
			Name:        assetName,
			Fingerprint: common.NewAssetFingerprint(policyID, assetName).String(),
		}).
		FirstOrCreate(&multiAsset).Error; err != nil {
		return 0, fmt.Errorf("failed to create multi-asset: %w", err)
	}

	// Cache the new ID
	mac.mutex.Lock()
	mac.cache[cacheKey] = multiAsset.ID
	mac.mutex.Unlock()

	return multiAsset.ID, nil
}

// NewAssetProcessor creates a new asset processor
func NewAssetProcessor(db *gorm.DB) *AssetProcessor {
	return &AssetProcessor{
		db:              db,
		multiAssetCache: NewMultiAssetCache(db),
		errorCollector:  GetGlobalErrorCollector(),
	}
}

// ProcessTransactionMints processes minting operations from a transaction
func (ap *AssetProcessor) ProcessTransactionMints(ctx context.Context, tx *gorm.DB, txID uint64, transaction interface{}) error {
	// Try to extract mints using interface assertion
	switch txWithMint := transaction.(type) {
	case interface{ AssetMint() *common.MultiAsset[common.MultiAssetTypeMint] }:
		mint := txWithMint.AssetMint()
		if mint == nil {
			return nil
		}

		log.Printf("Processing minting operations for transaction %d", txID)

		// Process each policy in the mint
		for _, policyID := range mint.Policies() {
			policyIDBytes := policyID[:]
			assetNames := mint.Assets(policyID)
			for _, assetNameBytes := range assetNames {
				// Get the amount for this asset
				amount := mint.Asset(policyID, assetNameBytes)
				if err := ap.processMintOperation(tx, txID, policyIDBytes, assetNameBytes, int64(amount)); err != nil {
					log.Printf("[WARNING] Mint operation error: %v", err)
					continue
				}
			}
		}
		return nil

	default:
		// Transaction doesn't support minting or has none
		return nil
	}
}

// processMintOperation processes a single mint operation
func (ap *AssetProcessor) processMintOperation(tx *gorm.DB, txID uint64, policyID []byte, assetName []byte, amount int64) error {
	// Get or create multi-asset using transaction context
	multiAssetID, err := ap.multiAssetCache.GetOrCreateMultiAssetWithTx(tx, policyID, assetName)
	if err != nil {
		return fmt.Errorf("failed to get multi-asset: %w", err)
	}

	// Create mint record
	maTxMint := &models.MaTxMint{
		IdentID:  multiAssetID,
		Quantity:  amount,
		TxID:      txID,
	}

	if err := tx.Create(maTxMint).Error; err != nil {
		return fmt.Errorf("failed to create mint record: %w", err)
	}

	log.Printf("[OK] Processed mint: policy=%s, asset=%s, amount=%d",
		hex.EncodeToString(policyID)[:8]+"...",
		string(assetName),
		amount)

	return nil
}

// ProcessOutputAssets processes multi-assets in transaction outputs
func (ap *AssetProcessor) ProcessOutputAssets(ctx context.Context, tx *gorm.DB, txID uint64, outputIndex int, output interface{}) error {
	// Try to extract assets using interface assertion for Mary+ outputs
	switch outputWithAssets := output.(type) {
	case interface{ Assets() *common.MultiAsset[common.MultiAssetTypeOutput] }:
		assets := outputWithAssets.Assets()
		if assets == nil {
			return nil
		}

		// Find the corresponding tx_out record
		var txOut models.TxOut
		if err := tx.Where("tx_id = ? AND index = ?", txID, outputIndex).First(&txOut).Error; err != nil {
			return fmt.Errorf("failed to find tx_out for tx_id=%d, index=%d: %w", txID, outputIndex, err)
		}

		// Process each asset in the output
		for _, policyID := range assets.Policies() {
			policyIDBytes := policyID[:]
			assetNames := assets.Assets(policyID)
			for _, assetNameBytes := range assetNames {
				// Get the amount for this asset
				amount := assets.Asset(policyID, assetNameBytes)
				if err := ap.processOutputAsset(tx, txOut.ID, policyIDBytes, assetNameBytes, uint64(amount)); err != nil {
					log.Printf("[WARNING] Output asset error: %v", err)
					continue
				}
			}
		}
		return nil

	default:
		// Output doesn't have assets (Byron/early Shelley) or wrong type
		return nil
	}
}

// processOutputAsset processes a single asset in an output
func (ap *AssetProcessor) processOutputAsset(tx *gorm.DB, txOutID uint64, policyID []byte, assetName []byte, quantity uint64) error {
	// Get or create multi-asset using transaction context
	multiAssetID, err := ap.multiAssetCache.GetOrCreateMultiAssetWithTx(tx, policyID, assetName)
	if err != nil {
		return fmt.Errorf("failed to get multi-asset: %w", err)
	}

	// Create output asset record
	maTxOut := &models.MaTxOut{
		IdentID:  multiAssetID,
		Quantity: quantity,
		TxOutID:   txOutID,
	}

	if err := tx.Create(maTxOut).Error; err != nil {
		return fmt.Errorf("failed to create output asset record: %w", err)
	}

	return nil
}

// ProcessTransactionAssetsBatch processes all assets for a transaction using pre-loaded output records (optimized)
func (ap *AssetProcessor) ProcessTransactionAssetsBatch(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, outputs []ledger.TransactionOutput, outputRecords []models.TxOut) error {
	// Performance logging - track optimization benefits
	outputCount := len(outputs)
	if outputCount > 0 {
		log.Printf("OPTIMIZED: Processing %d outputs with single batch query (saved %d individual queries)", outputCount, outputCount-1)
	}

	// Process minting operations
	if err := ap.ProcessTransactionMints(ctx, tx, txID, transaction); err != nil {
		return fmt.Errorf("failed to process mints: %w", err)
	}

	// Validate we have the expected number of output records
	if len(outputRecords) != outputCount {
		log.Printf("[WARNING] Output count mismatch: expected %d, got %d records for tx_id=%d", outputCount, len(outputRecords), txID)
	}

	// Create lookup map of output records by index for O(1) access
	outputMap := make(map[int]*models.TxOut)
	for i := range outputRecords {
		outputMap[int(outputRecords[i].Index)] = &outputRecords[i]
	}

	// Process assets in outputs using pre-loaded records
	assetsProcessed := 0
	for i, output := range outputs {
		if err := ap.ProcessOutputAssetsBatch(ctx, tx, txID, i, output, outputMap); err != nil {
			log.Printf("[WARNING] Failed to process output %d assets: %v", i, err)
			continue // Continue processing other outputs instead of failing entire transaction
		}
		assetsProcessed++
	}

	if assetsProcessed > 0 {
		log.Printf("[OK] Batch processed assets for %d/%d outputs in transaction %d", assetsProcessed, outputCount, txID)
	}

	return nil
}

// ProcessOutputAssetsBatch processes multi-assets in transaction outputs using pre-loaded output records
func (ap *AssetProcessor) ProcessOutputAssetsBatch(ctx context.Context, tx *gorm.DB, txID uint64, outputIndex int, output interface{}, outputMap map[int]*models.TxOut) error {
	// Try to extract assets using interface assertion for Mary+ outputs
	switch outputWithAssets := output.(type) {
	case interface{ Assets() *common.MultiAsset[common.MultiAssetTypeOutput] }:
		assets := outputWithAssets.Assets()
		if assets == nil {
			return nil
		}

		// Get the pre-loaded tx_out record from map (no database query!)
		txOut, exists := outputMap[outputIndex]
		if !exists {
			// Enhanced error handling - provide more context
			availableIndices := make([]int, 0, len(outputMap))
			for idx := range outputMap {
				availableIndices = append(availableIndices, idx)
			}
			return fmt.Errorf("pre-loaded tx_out not found for tx_id=%d, index=%d (available indices: %v)", txID, outputIndex, availableIndices)
		}

		// Count and process each asset in the output
		assetCount := 0
		for _, policyID := range assets.Policies() {
			policyIDBytes := policyID[:]
			assetNames := assets.Assets(policyID)
			for _, assetNameBytes := range assetNames {
				// Get the amount for this asset
				amount := assets.Asset(policyID, assetNameBytes)
				if err := ap.processOutputAsset(tx, txOut.ID, policyIDBytes, assetNameBytes, uint64(amount)); err != nil {
					log.Printf("[WARNING] Output asset error for tx_id=%d, output=%d: %v", txID, outputIndex, err)
					continue
				}
				assetCount++
			}
		}

		if assetCount > 0 {
			log.Printf(" Processed %d assets in output %d (tx_id=%d) using batch optimization", assetCount, outputIndex, txID)
		}
		
		return nil

	default:
		// Output doesn't have assets (Byron/early Shelley) or wrong type
		return nil
	}
}

// ProcessNativeAssets is a convenience method that processes both mints and outputs
func (ap *AssetProcessor) ProcessNativeAssets(ctx context.Context, tx *gorm.DB, txID uint64, transaction ledger.Transaction, outputs []ledger.TransactionOutput) error {
	// Process minting operations
	if err := ap.ProcessTransactionMints(ctx, tx, txID, transaction); err != nil {
		return fmt.Errorf("failed to process mints: %w", err)
	}

	// Process assets in outputs
	for i, output := range outputs {
		if err := ap.ProcessOutputAssets(ctx, tx, txID, i, output); err != nil {
			return fmt.Errorf("failed to process output %d assets: %w", i, err)
		}
	}

	return nil
}

// ValidateAsset performs basic validation on asset properties
func (ap *AssetProcessor) ValidateAsset(policyID []byte, assetName []byte) error {
	// Policy ID should be 28 bytes (Blake2b224 hash)
	if len(policyID) != 28 {
		return fmt.Errorf("invalid policy ID length: expected 28, got %d", len(policyID))
	}

	// Asset name should not exceed 32 bytes
	if len(assetName) > 32 {
		return fmt.Errorf("asset name too long: maximum 32 bytes, got %d", len(assetName))
	}

	return nil
}

// GetAssetFingerprint calculates the CIP-14 asset fingerprint
func (ap *AssetProcessor) GetAssetFingerprint(policyID []byte, assetName []byte) string {
	return common.NewAssetFingerprint(policyID, assetName).String()
}

// ClearCache clears the multi-asset cache
func (ap *AssetProcessor) ClearCache() {
	ap.multiAssetCache.mutex.Lock()
	defer ap.multiAssetCache.mutex.Unlock()
	ap.multiAssetCache.cache = make(map[string]uint64)
}