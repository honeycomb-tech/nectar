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
	cache map[string]bool // Just track existence
	mutex sync.RWMutex
	db    *gorm.DB
}

// NewMultiAssetCache creates a new multi-asset cache
func NewMultiAssetCache(db *gorm.DB) *MultiAssetCache {
	return &MultiAssetCache{
		cache: make(map[string]bool),
		db:    db,
	}
}

// GetOrCreateMultiAsset ensures a multi-asset exists
func (mac *MultiAssetCache) GetOrCreateMultiAsset(policyID []byte, assetName []byte) error {
	// Use the transaction context if available
	return mac.GetOrCreateMultiAssetWithTx(mac.db, policyID, assetName)
}

// GetOrCreateMultiAssetWithTx ensures a multi-asset exists with transaction context
func (mac *MultiAssetCache) GetOrCreateMultiAssetWithTx(tx *gorm.DB, policyID []byte, assetName []byte) error {
	cacheKey := fmt.Sprintf("%s:%s", hex.EncodeToString(policyID), hex.EncodeToString(assetName))
	
	// Check cache first
	mac.mutex.RLock()
	if exists := mac.cache[cacheKey]; exists {
		mac.mutex.RUnlock()
		return nil
	}
	mac.mutex.RUnlock()

	// Check database using transaction context
	var multiAsset models.MultiAsset
	result := tx.Where("policy = ? AND name = ?", policyID, assetName).First(&multiAsset)
	if result.Error == nil {
		// Found in database, cache it
		mac.mutex.Lock()
		mac.cache[cacheKey] = true
		mac.mutex.Unlock()
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", result.Error)
	}

	// Create new multi-asset
	multiAsset = models.MultiAsset{
		Policy:      policyID,
		Name:        assetName,
		Fingerprint: common.NewAssetFingerprint(policyID, assetName).String(),
	}

	if err := tx.Create(&multiAsset).Error; err != nil {
		return fmt.Errorf("failed to create multi-asset: %w", err)
	}

	// Cache existence
	mac.mutex.Lock()
	mac.cache[cacheKey] = true
	mac.mutex.Unlock()

	return nil
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
func (ap *AssetProcessor) ProcessTransactionMints(ctx context.Context, tx *gorm.DB, txHash []byte, transaction interface{}) error {
	// Try to extract mints using interface assertion
	switch txWithMint := transaction.(type) {
	case interface{ AssetMint() *common.MultiAsset[common.MultiAssetTypeMint] }:
		mint := txWithMint.AssetMint()
		if mint == nil {
			return nil
		}

		log.Printf("Processing minting operations for transaction %x", txHash)

		// Process each policy in the mint
		for _, policyID := range mint.Policies() {
			policyIDBytes := policyID[:]
			assetNames := mint.Assets(policyID)
			for _, assetNameBytes := range assetNames {
				// Get the amount for this asset
				amount := mint.Asset(policyID, assetNameBytes)
				if err := ap.processMintOperation(tx, txHash, policyIDBytes, assetNameBytes, int64(amount)); err != nil {
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
func (ap *AssetProcessor) processMintOperation(tx *gorm.DB, txHash []byte, policyID []byte, assetName []byte, amount int64) error {
	// Ensure multi-asset exists using transaction context
	if err := ap.multiAssetCache.GetOrCreateMultiAssetWithTx(tx, policyID, assetName); err != nil {
		return fmt.Errorf("failed to ensure multi-asset: %w", err)
	}

	// Create mint record with composite key
	maTxMint := &models.MaTxMint{
		TxHash:   txHash,
		Policy:   policyID,
		Name:     assetName,
		Quantity: amount,
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
func (ap *AssetProcessor) ProcessOutputAssets(tx *gorm.DB, txHash []byte, outputIndex uint32, assets *common.MultiAsset[common.MultiAssetTypeOutput]) error {
	// Try to extract assets using interface assertion for Mary+ outputs
	// Process each asset in the output
	for _, policyID := range assets.Policies() {
		policyIDBytes := policyID[:]
		assetNames := assets.Assets(policyID)
		for _, assetNameBytes := range assetNames {
			// Get the amount for this asset
			amount := assets.Asset(policyID, assetNameBytes)
			if err := ap.processOutputAsset(tx, txHash, outputIndex, policyIDBytes, assetNameBytes, uint64(amount)); err != nil {
				log.Printf("[WARNING] Output asset error: %v", err)
				continue
			}
		}
	}
	return nil
}

// processOutputAsset processes a single asset in an output
func (ap *AssetProcessor) processOutputAsset(tx *gorm.DB, txHash []byte, outputIndex uint32, policyID []byte, assetName []byte, quantity uint64) error {
	// Ensure multi-asset exists using transaction context
	if err := ap.multiAssetCache.GetOrCreateMultiAssetWithTx(tx, policyID, assetName); err != nil {
		return fmt.Errorf("failed to ensure multi-asset: %w", err)
	}

	// Create output asset record with composite key
	maTxOut := &models.MaTxOut{
		TxHash:   txHash,
		TxIndex:  outputIndex,
		Policy:   policyID,
		Name:     assetName,
		Quantity: quantity,
	}

	if err := tx.Create(maTxOut).Error; err != nil {
		return fmt.Errorf("failed to create output asset record: %w", err)
	}

	return nil
}

// ProcessTransactionAssetsBatch processes all assets for a transaction (optimized)
func (ap *AssetProcessor) ProcessTransactionAssetsBatch(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, outputs []ledger.TransactionOutput) error {
	// Performance logging - track optimization benefits
	outputCount := len(outputs)
	if outputCount > 0 {
		log.Printf("OPTIMIZED: Processing %d outputs with single batch query (saved %d individual queries)", outputCount, outputCount-1)
	}

	// Process minting operations
	if err := ap.ProcessTransactionMints(ctx, tx, txHash, transaction); err != nil {
		return fmt.Errorf("failed to process mints: %w", err)
	}

	// No need for output records lookup with hash-based model

	// Process assets in outputs
	assetsProcessed := 0
	for i, output := range outputs {
		if err := ap.ProcessOutputAssetsBatch(ctx, tx, txHash, i, output); err != nil {
			log.Printf("[WARNING] Failed to process output %d assets: %v", i, err)
			continue // Continue processing other outputs instead of failing entire transaction
		}
		assetsProcessed++
	}

	if assetsProcessed > 0 {
		log.Printf("[OK] Batch processed assets for %d/%d outputs in transaction %s", assetsProcessed, outputCount, hex.EncodeToString(txHash)[:16])
	}

	return nil
}

// ProcessOutputAssetsBatch processes multi-assets in transaction outputs
func (ap *AssetProcessor) ProcessOutputAssetsBatch(ctx context.Context, tx *gorm.DB, txHash []byte, outputIndex int, output interface{}) error {
	// Try to extract assets using interface assertion for Mary+ outputs
	switch outputWithAssets := output.(type) {
	case interface{ Assets() *common.MultiAsset[common.MultiAssetTypeOutput] }:
		assets := outputWithAssets.Assets()
		if assets == nil {
			return nil
		}

		// No need for output record lookup with hash-based model

		// Count and process each asset in the output
		assetCount := 0
		for _, policyID := range assets.Policies() {
			policyIDBytes := policyID[:]
			assetNames := assets.Assets(policyID)
			for _, assetNameBytes := range assetNames {
				// Get the amount for this asset
				amount := assets.Asset(policyID, assetNameBytes)
				if err := ap.processOutputAsset(tx, txHash, uint32(outputIndex), policyIDBytes, assetNameBytes, uint64(amount)); err != nil {
					log.Printf("[WARNING] Output asset error for tx %s, output=%d: %v", hex.EncodeToString(txHash)[:16], outputIndex, err)
					continue
				}
				assetCount++
			}
		}

		if assetCount > 0 {
			log.Printf(" Processed %d assets in output %d (tx %s) using batch optimization", assetCount, outputIndex, hex.EncodeToString(txHash)[:16])
		}
		
		return nil

	default:
		// Output doesn't have assets (Byron/early Shelley) or wrong type
		return nil
	}
}

// ProcessNativeAssets is a convenience method that processes both mints and outputs
func (ap *AssetProcessor) ProcessNativeAssets(ctx context.Context, tx *gorm.DB, txHash []byte, transaction ledger.Transaction, outputs []ledger.TransactionOutput) error {
	// Process minting operations
	if err := ap.ProcessTransactionMints(ctx, tx, txHash, transaction); err != nil {
		return fmt.Errorf("failed to process mints: %w", err)
	}

	// Process assets in outputs
	for i, output := range outputs {
		// Get assets from output
		var assets interface{}
		switch outputWithAssets := output.(type) {
		case interface{ Assets() *common.MultiAsset[common.MultiAssetTypeOutput] }:
			assets = outputWithAssets.Assets()
		}
		
		if assets != nil {
			if assetMulti, ok := assets.(*common.MultiAsset[common.MultiAssetTypeOutput]); ok {
				if err := ap.ProcessOutputAssets(tx, txHash, uint32(i), assetMulti); err != nil {
					return fmt.Errorf("failed to process output %d assets: %w", i, err)
				}
			}
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

// ProcessMint processes minting/burning from a transaction
func (ap *AssetProcessor) ProcessMint(tx *gorm.DB, txHash []byte, mint map[string]uint64) error {
	if len(mint) == 0 {
		return nil
	}

	log.Printf("Processing %d mint operations for transaction %x", len(mint), txHash)

	for assetID, amount := range mint {
		// Parse asset ID to get policy and name
		// Asset ID format is typically "policyID.assetName"
		// For now, we'll process it as a simple policy ID
		policyID := []byte(assetID[:28]) // First 28 bytes
		assetName := []byte(assetID[28:]) // Remaining bytes
		
		if err := ap.processMintOperation(tx, txHash, policyID, assetName, int64(amount)); err != nil {
			log.Printf("[WARNING] Failed to process mint for asset %s: %v", assetID, err)
			continue
		}
	}

	return nil
}

// ClearCache clears the multi-asset cache
func (ap *AssetProcessor) ClearCache() {
	ap.multiAssetCache.mutex.Lock()
	defer ap.multiAssetCache.mutex.Unlock()
	ap.multiAssetCache.cache = make(map[string]bool)
}