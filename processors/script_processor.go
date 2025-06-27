package processors

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	unifiederrors "nectar/errors"
	"nectar/models"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"golang.org/x/crypto/blake2b"
	"gorm.io/gorm"
)

// ScriptProcessor handles processing scripts for Alonzo era and beyond
type ScriptProcessor struct {
	db             *gorm.DB
	scriptCache    *ScriptCache
	datumCache     *DatumCache
	errorCollector *ErrorCollector
}

// ScriptCache manages script lookups with LRU eviction
type ScriptCache struct {
	cache *GenericLRUCache
	db    *gorm.DB
}

// NewScriptCache creates a new script cache with LRU eviction
func NewScriptCache(db *gorm.DB) *ScriptCache {
	// 100k entries should be enough for most scripts
	return &ScriptCache{
		cache: NewGenericLRUCache(100000),
		db:    db,
	}
}

// EnsureScript ensures a script exists
func (sc *ScriptCache) EnsureScript(tx *gorm.DB, txHash []byte, scriptHash []byte, scriptType string, scriptBytes []byte, scriptJson *string) error {
	hashHex := hex.EncodeToString(scriptHash)

	// Check cache first
	if _, exists := sc.cache.Get(hashHex); exists {
		return nil
	}

	// Check database
	var script models.Script
	result := tx.Where("hash = ?", scriptHash).First(&script)
	if result.Error == nil {
		// Found in database, cache it
		sc.cache.Put(hashHex, true)
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", result.Error)
	}

	// Create new script
	script = models.Script{
		TxHash: txHash,
		Hash:   scriptHash,
		Type:   scriptType,
		Bytes:  scriptBytes,
		Json:   scriptJson,
	}

	if err := tx.Create(&script).Error; err != nil {
		return fmt.Errorf("failed to create script: %w", err)
	}
	
	// Script saved successfully

	// Cache existence
	sc.cache.Put(hashHex, true)

	return nil
}

// ProcessInlineDatums processes inline datums from transaction outputs
func (sp *ScriptProcessor) ProcessInlineDatums(ctx context.Context, tx *gorm.DB, txHash []byte, transaction interface{}) error {
	// Get outputs from transaction
	var outputs []interface{}
	switch t := transaction.(type) {
	case interface{ Outputs() []interface{} }:
		outputs = t.Outputs()
	default:
		// Try to extract using common transaction interface
		if ledgerTx, ok := transaction.(common.Transaction); ok {
			txOutputs := ledgerTx.Outputs()
			outputs = make([]interface{}, len(txOutputs))
			for i, out := range txOutputs {
				outputs[i] = out
			}
		}
	}

	if len(outputs) == 0 {
		return nil
	}

	// Process each output for inline datums
	for _, output := range outputs {
		// Check if output has Datum() method
		if outputWithDatum, ok := output.(interface{ Datum() *cbor.LazyValue }); ok {
			datum := outputWithDatum.Datum()
			if datum != nil && datum.Cbor() != nil {
				// Calculate datum hash
				datumBytes := datum.Cbor()
				h, _ := blake2b.New256(nil)
				h.Write(datumBytes)
				datumHash := h.Sum(nil)

				// Store the datum using the cache
				if err := sp.datumCache.EnsureDatum(tx, txHash, datumHash, datumBytes); err != nil {
					unifiederrors.Get().Warning("ScriptProcessor", "StoreInlineDatum", fmt.Sprintf("Failed to store inline datum: %v", err))
				}
			}
		}
	}

	return nil
}

// DatumCache manages datum lookups with LRU eviction
type DatumCache struct {
	cache *GenericLRUCache
	db    *gorm.DB
}

// NewDatumCache creates a new datum cache with LRU eviction
func NewDatumCache(db *gorm.DB) *DatumCache {
	// 100k entries should be enough for most datums
	return &DatumCache{
		cache: NewGenericLRUCache(100000),
		db:    db,
	}
}

// EnsureDatum ensures a datum exists
func (dc *DatumCache) EnsureDatum(tx *gorm.DB, txHash []byte, datumHash []byte, datumBytes []byte) error {
	hashHex := hex.EncodeToString(datumHash)

	// Check cache first
	if _, exists := dc.cache.Get(hashHex); exists {
		return nil
	}

	// Check database
	var datum models.Datum
	result := tx.Where("hash = ?", datumHash).First(&datum)
	if result.Error == nil {
		// Found in database, cache it
		dc.cache.Put(hashHex, true)
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("database error: %w", result.Error)
	}

	// Create new datum
	datum = models.Datum{
		TxHash: txHash,
		Hash:   datumHash,
		Value:  datumBytes,
		Bytes:  datumBytes,
	}

	if err := tx.Create(&datum).Error; err != nil {
		return fmt.Errorf("failed to create datum: %w", err)
	}

	// Cache existence
	dc.cache.Put(hashHex, true)

	return nil
}

// NewScriptProcessor creates a new script processor
func NewScriptProcessor(db *gorm.DB) *ScriptProcessor {
	log.Printf("[SCRIPT_PROCESSOR] Creating new ScriptProcessor instance")
	return &ScriptProcessor{
		db:             db,
		scriptCache:    NewScriptCache(db),
		datumCache:     NewDatumCache(db),
		errorCollector: GetGlobalErrorCollector(),
	}
}

// ProcessTransactionScripts processes scripts from a transaction witness set
func (sp *ScriptProcessor) ProcessTransactionScripts(ctx context.Context, tx *gorm.DB, txHash []byte, witnessSet interface{}) error {
	if witnessSet == nil {
		return nil
	}
	
	// Debug: Check what's in the witness set
	if ws, ok := witnessSet.(interface{ PlutusV1Scripts() [][]byte }); ok {
		scripts := ws.PlutusV1Scripts()
		if len(scripts) > 0 {
			log.Printf("[SCRIPT_FOUND] Found %d PlutusV1 scripts in tx %x", len(scripts), txHash)
		}
	}
	if ws, ok := witnessSet.(interface{ NativeScripts() []common.NativeScript }); ok {
		scripts := ws.NativeScripts()
		if len(scripts) > 0 {
			log.Printf("[SCRIPT_FOUND] Found %d Native scripts in tx %x", len(scripts), txHash)
		}
	}
	if ws, ok := witnessSet.(interface{ PlutusV2Scripts() [][]byte }); ok {
		scripts := ws.PlutusV2Scripts()
		if len(scripts) > 0 {
			log.Printf("[SCRIPT_FOUND] Found %d PlutusV2 scripts in tx %x", len(scripts), txHash)
		}
	}

	// Process native scripts
	if err := sp.ProcessNativeScripts(ctx, tx, txHash, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessNativeScripts",
			fmt.Sprintf("failed to process native scripts: %v", err),
			fmt.Sprintf("tx_hash:%x", txHash))
	}

	// Process Plutus V1 scripts
	if err := sp.ProcessPlutusV1Scripts(ctx, tx, txHash, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessPlutusV1Scripts",
			fmt.Sprintf("failed to process PlutusV1 scripts: %v", err),
			fmt.Sprintf("tx_hash:%x", txHash))
	}

	// Process Plutus V2 scripts
	if err := sp.ProcessPlutusV2Scripts(ctx, tx, txHash, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessPlutusV2Scripts",
			fmt.Sprintf("failed to process PlutusV2 scripts: %v", err),
			fmt.Sprintf("tx_hash:%x", txHash))
	}

	// Process Plutus V3 scripts
	if err := sp.ProcessPlutusV3Scripts(ctx, tx, txHash, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessPlutusV3Scripts",
			fmt.Sprintf("failed to process PlutusV3 scripts: %v", err),
			fmt.Sprintf("tx_hash:%x", txHash))
	}

	return nil
}

// ProcessNativeScripts processes native scripts from witness set
func (sp *ScriptProcessor) ProcessNativeScripts(ctx context.Context, tx *gorm.DB, txHash []byte, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ NativeScripts() []common.NativeScript }:
		scripts := ws.NativeScripts()

		if len(scripts) > 0 {
			log.Printf("Processing %d native scripts for tx_hash=%x", len(scripts), txHash)
		}

		for i, script := range scripts {
			if err := sp.processScript(tx, txHash, script, "native", nil); err != nil {
				log.Printf("Warning: failed to process native script %d: %v", i+1, err)
				continue
			}
		}
	default:
		// Try reflection to see what methods are available
	}
	return nil
}

// ProcessPlutusV1Scripts processes PlutusV1 scripts from witness set
func (sp *ScriptProcessor) ProcessPlutusV1Scripts(ctx context.Context, tx *gorm.DB, txHash []byte, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ PlutusV1Scripts() [][]byte }:
		scripts := ws.PlutusV1Scripts()
		if len(scripts) > 0 {
		}
		for _, script := range scripts {
			if err := sp.processScript(tx, txHash, script, "plutus_v1", nil); err != nil {
				unifiederrors.Get().Warning("ScriptProcessor", "ProcessPlutusV1", fmt.Sprintf("Failed to process PlutusV1 script: %v", err))
				continue
			}
		}
	}
	return nil
}

// ProcessPlutusV2Scripts processes PlutusV2 scripts from witness set
func (sp *ScriptProcessor) ProcessPlutusV2Scripts(ctx context.Context, tx *gorm.DB, txHash []byte, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ PlutusV2Scripts() [][]byte }:
		scripts := ws.PlutusV2Scripts()
		if len(scripts) > 0 {
		}
		for _, script := range scripts {
			if err := sp.processScript(tx, txHash, script, "plutus_v2", nil); err != nil {
				unifiederrors.Get().Warning("ScriptProcessor", "ProcessPlutusV2", fmt.Sprintf("Failed to process PlutusV2 script: %v", err))
				continue
			}
		}
	}
	return nil
}

// ProcessPlutusV3Scripts processes PlutusV3 scripts from witness set
func (sp *ScriptProcessor) ProcessPlutusV3Scripts(ctx context.Context, tx *gorm.DB, txHash []byte, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ PlutusV3Scripts() [][]byte }:
		scripts := ws.PlutusV3Scripts()
		for _, script := range scripts {
			if err := sp.processScript(tx, txHash, script, "plutus_v3", nil); err != nil {
				log.Printf("[WARNING] Failed to process PlutusV3 script: %v", err)
				continue
			}
		}
	}
	return nil
}

// processScript processes a single script
func (sp *ScriptProcessor) processScript(tx *gorm.DB, txHash []byte, script interface{}, scriptType string, additionalData map[string]interface{}) error {
	
	var scriptBytes []byte
	var scriptHash []byte
	var scriptJson *string

	// Extract script bytes
	switch s := script.(type) {
	case []byte:
		scriptBytes = s
	case common.NativeScript:
		// For native scripts, get the actual script item and encode it
		scriptItem := s.Item()
		if encoded, err := cbor.Encode(scriptItem); err == nil {
			scriptBytes = encoded
		} else {
			return fmt.Errorf("cannot encode native script item: %w", err)
		}
	case interface{ Bytes() []byte }:
		scriptBytes = s.Bytes()
	case interface{ Cbor() []byte }:
		scriptBytes = s.Cbor()
	default:
		// Try to encode as CBOR
		if encoded, err := cbor.Encode(script); err == nil {
			scriptBytes = encoded
		} else {
			return fmt.Errorf("cannot extract script bytes from type %T", script)
		}
	}

	// Calculate script hash
	if scriptHash == nil && scriptBytes != nil {
		hasher, err := blake2b.New(28, nil)
		if err != nil {
			return fmt.Errorf("failed to create hasher: %w", err)
		}
		hasher.Write(scriptBytes)
		scriptHash = hasher.Sum(nil)
	}

	// Try to convert to JSON for debugging
	if jsonBytes, err := cbor.Encode(script); err == nil {
		jsonString := hex.EncodeToString(jsonBytes)
		scriptJson = &jsonString
	}

	// Store the script
	if err := sp.scriptCache.EnsureScript(tx, txHash, scriptHash, scriptType, scriptBytes, scriptJson); err != nil {
		return fmt.Errorf("failed to store script: %w", err)
	}

	// Successfully processed script
	return nil
}

// ProcessRedeemers processes redeemers from a transaction
func (sp *ScriptProcessor) ProcessRedeemers(ctx context.Context, tx *gorm.DB, txHash []byte, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface {
		Redeemers() common.TransactionWitnessRedeemers
	}:
		redeemers := ws.Redeemers()
		if redeemers == nil {
			return nil // No redeemers (e.g., in Shelley era)
		}

		// Process redeemers for each tag type
		for _, tag := range []common.RedeemerTag{
			common.RedeemerTagSpend,
			common.RedeemerTagMint,
			common.RedeemerTagCert,
			common.RedeemerTagReward,
		} {
			indexes := redeemers.Indexes(tag)
			for _, index := range indexes {
				if err := sp.processRedeemerByTagAndIndex(tx, txHash, tag, index, redeemers); err != nil {
					sp.errorCollector.ProcessingWarning("ScriptProcessor", "processRedeemer",
						fmt.Sprintf("failed to process redeemer: %v", err),
						fmt.Sprintf("tx_hash:%x,tag:%d,index:%d", txHash, tag, index))
					continue
				}
			}
		}
	}
	return nil
}

// processRedeemerByTagAndIndex processes a single redeemer by tag and index
func (sp *ScriptProcessor) processRedeemerByTagAndIndex(tx *gorm.DB, txHash []byte, tag common.RedeemerTag, index uint, redeemers common.TransactionWitnessRedeemers) error {
	// Get redeemer data and execution units
	data, exUnits := redeemers.Value(index, tag)

	// Convert tag to purpose string
	var purpose string
	switch tag {
	case common.RedeemerTagSpend:
		purpose = "spend"
	case common.RedeemerTagMint:
		purpose = "mint"
	case common.RedeemerTagCert:
		purpose = "cert"
	case common.RedeemerTagReward:
		purpose = "reward"
	default:
		purpose = "unknown"
	}

	// Get redeemer data bytes
	var redeemerData []byte
	if data.Cbor() != nil {
		redeemerData = data.Cbor()
	}

	// Calculate redeemer data hash
	var dataHash []byte
	if redeemerData != nil {
		hasher, err := blake2b.New256(nil)
		if err != nil {
			return fmt.Errorf("failed to create hasher: %w", err)
		}
		hasher.Write(redeemerData)
		dataHash = hasher.Sum(nil)
	}

	// Create redeemer data record
	redeemerDataRecord := &models.RedeemerData{
		Hash:   dataHash,
		TxHash: txHash,
		Value:  redeemerData,
		Bytes:  redeemerData,
	}

	if err := tx.Create(redeemerDataRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer data: %w", err)
	}

	// Create redeemer record
	redeemerHash := models.GenerateRedeemerHash(txHash, purpose, uint32(index))
	redeemerRecord := &models.Redeemer{
		Hash:             redeemerHash,
		TxHash:           txHash,
		Purpose:          purpose,
		Index:            uint32(index),
		RedeemerDataHash: dataHash,
		ScriptHash:       nil, // Would need to be linked to actual script
		UnitMem:          exUnits.Memory,
		UnitSteps:        exUnits.Steps,
		Fee:              nil, // Fee calculation would require protocol parameters
	}

	if err := tx.Create(redeemerRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer: %w", err)
	}

	// Successfully processed redeemer
	return nil
}

// processRedeemer processes a single redeemer (legacy method)
func (sp *ScriptProcessor) processRedeemer(tx *gorm.DB, txHash []byte, index int, redeemer interface{}) error {
	// Extract redeemer data
	var redeemerData []byte
	var purpose string = "spend"             // Default to most common purpose
	var redeemerIndex uint32 = uint32(index) // Default to passed index

	// Try to extract redeemer data
	switch r := redeemer.(type) {
	case interface{ Data() []byte }:
		redeemerData = r.Data()
	case interface{ Cbor() []byte }:
		redeemerData = r.Cbor()
	}

	// Try to extract purpose
	if r, ok := redeemer.(interface{ Purpose() string }); ok {
		purpose = r.Purpose()
	} else if r, ok := redeemer.(interface{ Tag() uint8 }); ok {
		// Map tag to purpose: 0=spend, 1=mint, 2=cert, 3=reward
		switch r.Tag() {
		case 0:
			purpose = "spend"
		case 1:
			purpose = "mint"
		case 2:
			purpose = "cert"
		case 3:
			purpose = "reward"
		}
	}

	// Try to extract index
	if r, ok := redeemer.(interface{ Index() uint32 }); ok {
		redeemerIndex = r.Index()
	}

	// Extract execution units from redeemer
	var unitMem uint64 = 1000000 // Default values
	var unitSteps uint64 = 500000

	// Note: Execution units extraction depends on the redeemer type
	// The interface{} doesn't guarantee the ExUnits method exists
	// Keep default values for now

	// Calculate redeemer data hash
	var dataHash []byte
	if redeemerData != nil {
		hasher, err := blake2b.New256(nil)
		if err != nil {
			return fmt.Errorf("failed to create hasher: %w", err)
		}
		hasher.Write(redeemerData)
		dataHash = hasher.Sum(nil)
	}

	// Create redeemer data record
	redeemerDataRecord := &models.RedeemerData{
		Hash:   dataHash,
		TxHash: txHash,
		Value:  redeemerData,
		Bytes:  redeemerData,
	}

	if err := tx.Create(redeemerDataRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer data: %w", err)
	}

	// Create redeemer record
	redeemerHash := models.GenerateRedeemerHash(txHash, purpose, redeemerIndex)
	redeemerRecord := &models.Redeemer{
		Hash:             redeemerHash,
		TxHash:           txHash,
		Purpose:          purpose,
		Index:            redeemerIndex,
		RedeemerDataHash: dataHash,
		ScriptHash:       nil, // Would need to be linked to actual script
		UnitMem:          unitMem,
		UnitSteps:        unitSteps,
		Fee:              nil, // Fee calculation would require protocol parameters
	}

	if err := tx.Create(redeemerRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer: %w", err)
	}

	return nil
}

// CalculateScriptDataHash calculates the script data hash for a transaction
func (sp *ScriptProcessor) CalculateScriptDataHash(redeemers []interface{}, datums []interface{}) ([]byte, error) {
	// This would implement the specific CDDL encoding required for script data hash
	// For now, return nil as this is complex and era-specific
	return nil, nil
}

// GetScriptStats returns statistics about scripts
func (sp *ScriptProcessor) GetScriptStats(tx *gorm.DB) (*ScriptStats, error) {
	var stats ScriptStats

	// Count scripts by type
	rows, err := tx.Model(&models.Script{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to count scripts: %w", err)
	}
	defer rows.Close()

	stats.ScriptsByType = make(map[string]int64)
	for rows.Next() {
		var scriptType string
		var count int64
		if err := rows.Scan(&scriptType, &count); err != nil {
			continue
		}
		stats.ScriptsByType[scriptType] = count
	}

	// Total scripts
	if err := tx.Model(&models.Script{}).Count(&stats.TotalScripts).Error; err != nil {
		return nil, fmt.Errorf("failed to count total scripts: %w", err)
	}

	// Total redeemers
	if err := tx.Model(&models.Redeemer{}).Count(&stats.TotalRedeemers).Error; err != nil {
		return nil, fmt.Errorf("failed to count redeemers: %w", err)
	}

	return &stats, nil
}

// ScriptStats contains script statistics
type ScriptStats struct {
	TotalScripts   int64
	TotalRedeemers int64
	ScriptsByType  map[string]int64
}

// ClearCache clears the script and datum caches
func (sp *ScriptProcessor) ClearCache() {
	sp.scriptCache.cache.Clear()
	sp.datumCache.cache.Clear()
}

// ProcessTransaction processes scripts and related data from a transaction
func (sp *ScriptProcessor) ProcessTransaction(tx *gorm.DB, txHash []byte, transaction interface{}, blockType uint) error {
	// Only log for smart contract eras
	if blockType >= 5 { // Alonzo or later
		
		// Check if it's a concrete type
		switch transaction.(type) {
		case *ledger.AlonzoTransaction:
		case ledger.AlonzoTransaction:
		case *ledger.BabbageTransaction:
		case ledger.BabbageTransaction:
		default:
		}
	}
	// Extract witness set if available
	var witnessSet interface{}
	
	// Try to get witness set based on transaction type
	switch tx := transaction.(type) {
	case *ledger.AlonzoTransaction:
		witnessSet = &tx.WitnessSet
		if blockType >= 5 {
			log.Printf("[ALONZO_TX] Got AlonzoTransaction with witness set for tx %x", txHash)
		}
	case *ledger.BabbageTransaction:
		witnessSet = &tx.WitnessSet
		if blockType >= 5 {
			log.Printf("[BABBAGE_TX] Got BabbageTransaction with witness set for tx %x", txHash)
		}
	case common.Transaction:
		// Try the interface method as fallback
		witnessSet = tx.Witnesses()
		if blockType >= 5 && witnessSet != nil {
			log.Printf("[COMMON_TX] Got witness set via interface for tx %x", txHash)
		}
	default:
		// Fall back to other methods
		switch txWithWitness := transaction.(type) {
		case interface{ WitnessSet() interface{} }:
			witnessSet = txWithWitness.WitnessSet()
		case interface{ Witnesses() interface{} }:
			witnessSet = txWithWitness.Witnesses()
		default:
			// For debugging, let's see what type we're actually getting
			log.Printf("[TRANSACTION_TYPE] No witness methods found. Transaction type: %T for tx %x", transaction, txHash)
		}
	}

	if witnessSet != nil {
		// Count transactions with witness sets in smart contract eras
		if blockType >= 5 {
			log.Printf("[WITNESS_FOUND] Transaction %x has witness set in era %d", txHash, blockType)
		}
	} else if blockType >= 5 {
		// Log when witness set is nil in smart contract eras
		log.Printf("[NO_WITNESS] Transaction %x has NO witness set in era %d", txHash, blockType)
		
		// Process scripts
		if err := sp.ProcessTransactionScripts(context.Background(), tx, txHash, witnessSet); err != nil {
			return fmt.Errorf("failed to process scripts: %w", err)
		}

		// Process redeemers
		if err := sp.ProcessRedeemers(context.Background(), tx, txHash, witnessSet); err != nil {
			return fmt.Errorf("failed to process redeemers: %w", err)
		}
	}

	// Process inline datums from outputs (Babbage+)
	if blockType >= 6 { // BlockTypeBabbage
		if err := sp.ProcessInlineDatums(context.Background(), tx, txHash, transaction); err != nil {
			return fmt.Errorf("failed to process inline datums: %w", err)
		}
	}

	return nil
}
