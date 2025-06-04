# BlockHeader Support Implementation

## Overview
This document describes the implementation of BlockHeader support in Nectar to align with gouroboros reference behavior.

## Problem Statement
Previously, Nectar's `chainSyncRollForwardHandler` only supported full `ledger.Block` objects and would fail with "invalid block data type" errors when receiving `ledger.BlockHeader` objects from the Cardano node.

This caused sync failures in scenarios where the node sends block headers instead of full blocks, which is common in certain sync modes and network conditions.

## Gouroboros Reference Pattern
Gouroboros handles both cases in its chainSyncRollForwardHandler:
- **Full Block**: Process directly
- **BlockHeader**: Use BlockFetch client to retrieve the full block using the header's slot and hash

## Implementation Details

### Before (Nectar)
```go
func (rai *ReferenceAlignedIndexer) chainSyncRollForwardHandler(
	ctx chainsync.CallbackContext,
	blockType uint,
	blockData any,
	tip chainsync.Tip,
) error {
	block, ok := blockData.(ledger.Block)
	if !ok {
		return fmt.Errorf("invalid block data type: %T", blockData)
	}
	return rai.processBlock(block, blockType)
}
```

### After (Nectar with BlockHeader Support)
```go
func (rai *ReferenceAlignedIndexer) chainSyncRollForwardHandler(
	ctx chainsync.CallbackContext,
	blockType uint,
	blockData any,
	tip chainsync.Tip,
) error {
	var block ledger.Block
	switch v := blockData.(type) {
	case ledger.Block:
		block = v
		log.Printf("📦 Received full block for slot %d (type %d)", block.SlotNumber(), blockType)
	case ledger.BlockHeader:
		blockSlot := v.SlotNumber()
		blockHash := v.Hash().Bytes()
		log.Printf("📋 Received block header for slot %d (type %d), fetching full block...", blockSlot, blockType)
		
		// Connection safety checks
		rai.connMutex.RLock()
		conn := rai.oConn
		rai.connMutex.RUnlock()
		
		if conn == nil {
			return fmt.Errorf("empty ouroboros connection, aborting")
		}
		
		// BlockFetch availability checks
		if !rai.useBlockFetch.Load() {
			log.Printf("❌ BlockFetch not available but received BlockHeader for slot %d", blockSlot)
			return fmt.Errorf("received BlockHeader but BlockFetch not available (Node-to-Client mode)")
		}
		
		if conn.BlockFetch() == nil || conn.BlockFetch().Client == nil {
			log.Printf("❌ BlockFetch client is nil for slot %d", blockSlot)
			return fmt.Errorf("BlockFetch client is not available")
		}
		
		// Fetch full block using BlockFetch
		var err error
		block, err = conn.BlockFetch().Client.GetBlock(common.NewPoint(blockSlot, blockHash))
		if err != nil {
			log.Printf("❌ Failed to fetch block from header (slot %d): %v", blockSlot, err)
			rai.addErrorEntry("BlockFetch", fmt.Sprintf("Failed to fetch block from header (slot %d): %v", blockSlot, err))
			return fmt.Errorf("failed to fetch block from header: %w", err)
		}
		
		log.Printf("✅ Successfully fetched full block for slot %d via BlockFetch", blockSlot)
		rai.addActivityEntry("BlockHeader", fmt.Sprintf("Fetched full block for slot %d via BlockFetch", blockSlot), map[string]interface{}{
			"slot": blockSlot,
			"hash": fmt.Sprintf("%x", blockHash),
		})
	default:
		return fmt.Errorf("invalid block data type: %T", blockData)
	}

	return rai.processBlock(block, blockType)
}
```

## Key Features Added

### 1. Type Switch Pattern
- Handles both `ledger.Block` and `ledger.BlockHeader` types
- Matches gouroboros reference implementation exactly

### 2. BlockFetch Integration
- Uses BlockFetch client to retrieve full blocks from headers
- Proper point creation using slot and hash from header

### 3. Safety Checks
- Validates connection availability
- Ensures BlockFetch is enabled (Node-to-Node mode)
- Verifies BlockFetch client is not nil

### 4. Enhanced Logging
- Debug logs for both Block and BlockHeader paths
- Success/failure logging for BlockFetch operations
- Activity feed integration for dashboard

### 5. Error Tracking
- Integrates with existing error statistics system
- Tracks BlockFetch failures for monitoring

## Benefits

### 1. Sync Reliability
- No more failures when node sends headers instead of blocks
- Handles various network conditions and sync modes

### 2. Reference Alignment
- Matches gouroboros behavior exactly
- Reduces sync discrepancies

### 3. Better Debugging
- Clear logging shows which path is taken
- Error tracking helps identify issues

### 4. Graceful Degradation
- Proper error messages for unsupported scenarios
- Falls back appropriately when BlockFetch unavailable

## Monitoring

### Log Messages to Watch For
- `📦 Received full block` - Normal block processing
- `📋 Received block header` - BlockHeader path activated  
- `✅ Successfully fetched full block via BlockFetch` - Successful header→block conversion
- `❌ Failed to fetch block from header` - BlockFetch failures

### Dashboard Integration
- Activity feed shows BlockHeader fetch operations
- Error statistics track BlockFetch failures
- Counters help monitor Block vs BlockHeader ratios

## Testing

### 1. Normal Operation
- Monitor logs for mix of Block and BlockHeader messages
- Verify no "invalid block data type" errors

### 2. Node-to-Client Mode
- Should see appropriate error messages for BlockHeaders
- Should continue working with full Blocks

### 3. BlockFetch Failures
- Error tracking should capture fetch failures
- Sync should stop gracefully rather than panic

## Compatibility

### Requirements
- Node-to-Node connection for BlockHeader support
- BlockFetch protocol enabled
- Compatible gouroboros library version

### Fallback Behavior
- Node-to-Client mode: BlockHeaders will error (expected)
- Missing BlockFetch: Clear error messages
- Connection issues: Proper error handling

## Next Steps

This implementation resolves the BlockHeader handling gap between Nectar and gouroboros. The next alignment areas to address are:

1. Processing architecture simplification
2. Network magic handling alignment  
3. Connection management simplification
4. Error handling patterns

## Verification Commands

After deployment, verify BlockHeader support with:
```bash
# Monitor for BlockHeader activity
tail -f nectar.log | grep -E "(📦|📋|BlockHeader)"

# Check error statistics
curl http://localhost:8080/api/errors | jq '.blockfetch_errors'

# Verify sync continues through mixed Block/BlockHeader scenarios
curl http://localhost:8080/api/stats | jq '.blocks_per_second'
```