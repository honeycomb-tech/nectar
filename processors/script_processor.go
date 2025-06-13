package processors

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"nectar/models"
	"sync"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"gorm.io/gorm"
	"golang.org/x/crypto/blake2b"
)

// ScriptProcessor handles processing scripts for Alonzo era and beyond
type ScriptProcessor struct {
	db             *gorm.DB
	scriptCache    *ScriptCache
	datumCache     *DatumCache
	errorCollector *ErrorCollector
}

// ScriptCache manages script lookups
type ScriptCache struct {
	cache map[string]uint64
	mutex sync.RWMutex
	db    *gorm.DB
}

// NewScriptCache creates a new script cache
func NewScriptCache(db *gorm.DB) *ScriptCache {
	return &ScriptCache{
		cache: make(map[string]uint64),
		db:    db,
	}
}

// GetOrCreateScript retrieves or creates a script ID
func (sc *ScriptCache) GetOrCreateScript(tx *gorm.DB, txID uint64, scriptHash []byte, scriptType string, scriptBytes []byte, scriptJson *string) (uint64, error) {
	hashHex := hex.EncodeToString(scriptHash)
	
	// Check cache first
	sc.mutex.RLock()
	if id, exists := sc.cache[hashHex]; exists {
		sc.mutex.RUnlock()
		return id, nil
	}
	sc.mutex.RUnlock()

	// Check database
	var script models.Script
	result := tx.Where("hash = ?", scriptHash).First(&script)
	if result.Error == nil {
		// Found in database, cache it
		sc.mutex.Lock()
		sc.cache[hashHex] = script.ID
		sc.mutex.Unlock()
		return script.ID, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("database error: %w", result.Error)
	}

	// Create new script
	script = models.Script{
		TxID:  txID,
		Hash:  scriptHash,
		Type:  scriptType,
		Bytes: scriptBytes,
		Json:  scriptJson,
	}

	if err := tx.Create(&script).Error; err != nil {
		return 0, fmt.Errorf("failed to create script: %w", err)
	}

	// Cache the new ID
	sc.mutex.Lock()
	sc.cache[hashHex] = script.ID
	sc.mutex.Unlock()

	return script.ID, nil
}

// DatumCache manages datum lookups
type DatumCache struct {
	cache map[string]uint64
	mutex sync.RWMutex
	db    *gorm.DB
}

// NewDatumCache creates a new datum cache
func NewDatumCache(db *gorm.DB) *DatumCache {
	return &DatumCache{
		cache: make(map[string]uint64),
		db:    db,
	}
}

// GetOrCreateDatum retrieves or creates a datum ID
func (dc *DatumCache) GetOrCreateDatum(tx *gorm.DB, txID uint64, datumHash []byte, datumBytes []byte) (uint64, error) {
	hashHex := hex.EncodeToString(datumHash)
	
	// Check cache first
	dc.mutex.RLock()
	if id, exists := dc.cache[hashHex]; exists {
		dc.mutex.RUnlock()
		return id, nil
	}
	dc.mutex.RUnlock()

	// Check database
	var datum models.Datum
	result := tx.Where("hash = ?", datumHash).First(&datum)
	if result.Error == nil {
		// Found in database, cache it
		dc.mutex.Lock()
		dc.cache[hashHex] = datum.ID
		dc.mutex.Unlock()
		return datum.ID, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("database error: %w", result.Error)
	}

	// Create new datum
	datum = models.Datum{
		TxID:  txID,
		Hash:  datumHash,
		Value: datumBytes,
		Bytes: datumBytes,
	}

	if err := tx.Create(&datum).Error; err != nil {
		return 0, fmt.Errorf("failed to create datum: %w", err)
	}

	// Cache the new ID
	dc.mutex.Lock()
	dc.cache[hashHex] = datum.ID
	dc.mutex.Unlock()

	return datum.ID, nil
}

// NewScriptProcessor creates a new script processor
func NewScriptProcessor(db *gorm.DB) *ScriptProcessor {
	return &ScriptProcessor{
		db:             db,
		scriptCache:    NewScriptCache(db),
		datumCache:     NewDatumCache(db),
		errorCollector: GetGlobalErrorCollector(),
	}
}

// ProcessTransactionScripts processes scripts from a transaction witness set
func (sp *ScriptProcessor) ProcessTransactionScripts(ctx context.Context, tx *gorm.DB, txID uint64, witnessSet interface{}) error {
	if witnessSet == nil {
		return nil
	}

	// Process native scripts
	if err := sp.ProcessNativeScripts(ctx, tx, txID, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessNativeScripts",
			fmt.Sprintf("failed to process native scripts: %v", err),
			fmt.Sprintf("tx_id:%d", txID))
	}

	// Process Plutus V1 scripts
	if err := sp.ProcessPlutusV1Scripts(ctx, tx, txID, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessPlutusV1Scripts",
			fmt.Sprintf("failed to process PlutusV1 scripts: %v", err),
			fmt.Sprintf("tx_id:%d", txID))
	}

	// Process Plutus V2 scripts
	if err := sp.ProcessPlutusV2Scripts(ctx, tx, txID, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessPlutusV2Scripts",
			fmt.Sprintf("failed to process PlutusV2 scripts: %v", err),
			fmt.Sprintf("tx_id:%d", txID))
	}

	// Process Plutus V3 scripts
	if err := sp.ProcessPlutusV3Scripts(ctx, tx, txID, witnessSet); err != nil {
		sp.errorCollector.ProcessingWarning("ScriptProcessor", "ProcessPlutusV3Scripts",
			fmt.Sprintf("failed to process PlutusV3 scripts: %v", err),
			fmt.Sprintf("tx_id:%d", txID))
	}

	return nil
}

// ProcessNativeScripts processes native scripts from witness set
func (sp *ScriptProcessor) ProcessNativeScripts(ctx context.Context, tx *gorm.DB, txID uint64, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ NativeScripts() []common.NativeScript }:
		scripts := ws.NativeScripts()
		
		if len(scripts) > 0 {
			log.Printf("Processing %d native scripts for tx_id=%d", len(scripts), txID)
		}
		
		for i, script := range scripts {
			if err := sp.processScript(tx, txID, script, "native", nil); err != nil {
				log.Printf("Warning: failed to process native script %d: %v", i+1, err)
				continue
			}
		}
	}
	return nil
}

// ProcessPlutusV1Scripts processes PlutusV1 scripts from witness set
func (sp *ScriptProcessor) ProcessPlutusV1Scripts(ctx context.Context, tx *gorm.DB, txID uint64, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ PlutusV1Scripts() [][]byte }:
		scripts := ws.PlutusV1Scripts()
		for _, script := range scripts {
			if err := sp.processScript(tx, txID, script, "plutus_v1", nil); err != nil {
				log.Printf("[WARNING] Failed to process PlutusV1 script: %v", err)
				continue
			}
		}
	}
	return nil
}

// ProcessPlutusV2Scripts processes PlutusV2 scripts from witness set
func (sp *ScriptProcessor) ProcessPlutusV2Scripts(ctx context.Context, tx *gorm.DB, txID uint64, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ PlutusV2Scripts() [][]byte }:
		scripts := ws.PlutusV2Scripts()
		for _, script := range scripts {
			if err := sp.processScript(tx, txID, script, "plutus_v2", nil); err != nil {
				log.Printf("[WARNING] Failed to process PlutusV2 script: %v", err)
				continue
			}
		}
	}
	return nil
}

// ProcessPlutusV3Scripts processes PlutusV3 scripts from witness set
func (sp *ScriptProcessor) ProcessPlutusV3Scripts(ctx context.Context, tx *gorm.DB, txID uint64, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ PlutusV3Scripts() [][]byte }:
		scripts := ws.PlutusV3Scripts()
		for _, script := range scripts {
			if err := sp.processScript(tx, txID, script, "plutus_v3", nil); err != nil {
				log.Printf("[WARNING] Failed to process PlutusV3 script: %v", err)
				continue
			}
		}
	}
	return nil
}

// processScript processes a single script
func (sp *ScriptProcessor) processScript(tx *gorm.DB, txID uint64, script interface{}, scriptType string, additionalData map[string]interface{}) error {
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
	_, err := sp.scriptCache.GetOrCreateScript(tx, txID, scriptHash, scriptType, scriptBytes, scriptJson)
	if err != nil {
		return fmt.Errorf("failed to store script: %w", err)
	}

	// Successfully processed script
	return nil
}

// ProcessRedeemers processes redeemers from a transaction
func (sp *ScriptProcessor) ProcessRedeemers(ctx context.Context, tx *gorm.DB, txID uint64, witnessSet interface{}) error {
	switch ws := witnessSet.(type) {
	case interface{ Redeemers() common.TransactionWitnessRedeemers }:
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
				if err := sp.processRedeemerByTagAndIndex(tx, txID, tag, index, redeemers); err != nil {
					sp.errorCollector.ProcessingWarning("ScriptProcessor", "processRedeemer",
						fmt.Sprintf("failed to process redeemer: %v", err),
						fmt.Sprintf("tx_id:%d,tag:%d,index:%d", txID, tag, index))
					continue
				}
			}
		}
	}
	return nil
}

// processRedeemerByTagAndIndex processes a single redeemer by tag and index
func (sp *ScriptProcessor) processRedeemerByTagAndIndex(tx *gorm.DB, txID uint64, tag common.RedeemerTag, index uint, redeemers common.TransactionWitnessRedeemers) error {
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
		Hash:  dataHash,
		TxID:  txID,
		Value: redeemerData,
		Bytes: redeemerData,
	}

	if err := tx.Create(redeemerDataRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer data: %w", err)
	}

	// Create redeemer record
	redeemerRecord := &models.Redeemer{
		TxID:       txID,
		Purpose:    purpose,
		Index:      uint32(index),
		DataID:     &redeemerDataRecord.ID,
		ScriptHash: nil,     // Would need to be linked to actual script
		UnitMem:    exUnits.Memory,
		UnitSteps:  exUnits.Steps,
		Fee:        nil,     // TODO: Calculate fee based on execution units
	}

	if err := tx.Create(redeemerRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer: %w", err)
	}

	// Successfully processed redeemer
	return nil
}

// processRedeemer processes a single redeemer (legacy method)
func (sp *ScriptProcessor) processRedeemer(tx *gorm.DB, txID uint64, index int, redeemer interface{}) error {
	// Extract redeemer data
	var redeemerData []byte
	var purpose string = "spend" // Default to most common purpose
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

	// TODO: Extract execution units (memory and steps) from redeemer
	// when the interface methods are available

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
		Hash:  dataHash,
		TxID:  txID,
		Value: redeemerData,
		Bytes: redeemerData,
	}

	if err := tx.Create(redeemerDataRecord).Error; err != nil {
		return fmt.Errorf("failed to create redeemer data: %w", err)
	}

	// Create redeemer record
	redeemerRecord := &models.Redeemer{
		TxID:       txID,
		Purpose:    purpose,
		Index:      redeemerIndex,
		DataID:    &redeemerDataRecord.ID,
		ScriptHash: nil,     // Would need to be linked to actual script
		UnitMem:    1000000, // TODO: Extract from redeemer ExUnits when available
		UnitSteps:  500000,  // TODO: Extract from redeemer ExUnits when available
		Fee:        nil,     // TODO: Calculate fee based on execution units
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
	sp.scriptCache.mutex.Lock()
	sp.scriptCache.cache = make(map[string]uint64)
	sp.scriptCache.mutex.Unlock()

	sp.datumCache.mutex.Lock()
	sp.datumCache.cache = make(map[string]uint64)
	sp.datumCache.mutex.Unlock()
}