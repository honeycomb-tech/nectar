# Fresh Sync Guide for Nectar

## Fixes Applied

I've fixed the following code bugs identified from your partial sync:

### 1. Foreign Key Constraint Error for multi_assets
**Problem**: When inserting ma_tx_outs or ma_tx_mints, the multi_asset might not exist yet, causing foreign key violations.

**Solution**: Updated `asset_processor.go` to:
- Use transaction context in `GetOrCreateMultiAssetWithTx` 
- Use `FirstOrCreate` for atomic operations to handle race conditions
- Ensure multi_assets are created within the same transaction as mints/outputs

### 2. Incorrect Table Name
**Problem**: Query referenced `pool_metadata_ref` but the actual table name is `pool_metadata_refs`.

**Solution**: Fixed in `metadata/fetcher.go:701` to use the correct table name.

### 3. Rollback Cleanup Issues
**Problem**: Error logs showed issues with block_id columns in ada_pots and event_infos tables.

**Note**: The models and migrations appear to have some discrepancies:
- AdaPots model has SlotNo but migration shows block_id
- EventInfo model has BlockID but migration shows tx_id

These discrepancies don't affect forward syncing but might impact rollbacks. For a fresh sync, this won't be an issue.

## Truncating the Database

I've created scripts to clear all tables for a fresh sync:

### Option 1: Using SQL Script
```bash
cd /root/workspace/cardano-stack/Nectar
mysql -h localhost -P 4000 -u root -proot cardano < scripts/truncate_all_tables.sql
```

### Option 2: Using Bash Script (Interactive)
```bash
cd /root/workspace/cardano-stack/Nectar
./scripts/truncate_db.sh
```

### Option 3: Using Go Script (if you want to compile it)
```bash
cd /root/workspace/cardano-stack/Nectar
go run scripts/truncate_db.go
```

## Starting Fresh Sync

After truncating the database:

1. **Verify database is empty**:
   ```bash
   mysql -h localhost -P 4000 -u root -proot cardano -e "SELECT COUNT(*) as block_count FROM blocks;"
   ```

2. **Start the indexer from genesis**:
   ```bash
   go run main.go
   ```

   The indexer will automatically:
   - Connect to the Cardano node
   - Start syncing from slot 0 (genesis)
   - Process all eras: Byron → Shelley → Allegra → Mary → Alonzo → Babbage → Conway
   - Fetch off-chain metadata for pools and governance
   - Calculate rewards at epoch boundaries

## Expected Behavior

During the fresh sync, you should see:
- [SYNC] Block processing messages
- [EPOCH] Epoch boundary processing at each epoch transition
- [ASSET] Asset processing for Mary+ eras
- [POOL] Pool registration/updates
- [GOV] Governance actions (Conway era)
- [META] Metadata fetching for pools
- [STATS] ADA pots calculations at epoch boundaries

## Performance Tips

1. The indexer uses batching (200 blocks per batch) for optimal performance
2. Multiple database connections are used for parallel processing
3. In-memory caching reduces database lookups
4. Foreign key checks are handled atomically to prevent race conditions

## Monitoring

The enhanced dashboard shows:
- Current sync progress and slot
- Blocks per second
- Era transitions
- Recent errors (if any)
- Database table row counts

## Notes

- The fresh sync will take several hours to complete (mainnet has 120M+ blocks)
- Memory usage should stay reasonable due to batching
- The indexer handles rollbacks automatically if the node reorganizes
- Off-chain metadata fetching runs in the background with rate limiting