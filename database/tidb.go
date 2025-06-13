package database

import (
	"fmt"
	"log"
	"nectar/errors"
	"nectar/models"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	DefaultTiDBDSN = "root:Ds*uS+pG278T@-3K60@tcp(127.0.0.1:4000)/nectar?charset=utf8mb4&parseTime=True&loc=Local&timeout=300s&readTimeout=300s&writeTimeout=300s&maxAllowedPacket=67108864&autocommit=true&tx_isolation='READ-COMMITTED'"
	NectarDBDSN    = "root:Ds*uS+pG278T@-3K60@tcp(127.0.0.1:4000)/nectar?charset=utf8mb4&parseTime=True&loc=Local&timeout=300s&readTimeout=300s&writeTimeout=300s&maxAllowedPacket=67108864&autocommit=true&tx_isolation='READ-COMMITTED'"
)

// InitTiDB initializes the TiDB connection with load balancer and optimizations
func InitTiDB() (*gorm.DB, error) {
	// Get DSN from environment or use default
	baseDSN := os.Getenv("TIDB_DSN")
	if baseDSN == "" {
		baseDSN = DefaultTiDBDSN
	}

	// First, connect without specifying a database (via HAProxy load balancer)
	baseDB, err := gorm.Open(mysql.Open(baseDSN), &gorm.Config{
		Logger: errors.NewGormLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TiDB cluster via load balancer: %w", err)
	}

	// Create the nectar database if it doesn't exist
	if err := baseDB.Exec("CREATE DATABASE IF NOT EXISTS nectar").Error; err != nil {
		return nil, fmt.Errorf("failed to create nectar database: %w", err)
	}

	// Close the base connection
	sqlDB, err := baseDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.Close()

	// Now connect to the nectar database (via HAProxy load balancer)
	nectarDSN := os.Getenv("NECTAR_DSN")
	if nectarDSN == "" {
		nectarDSN = NectarDBDSN
	}

	db, err := gorm.Open(mysql.Open(nectarDSN), &gorm.Config{
		Logger: errors.NewGormLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nectar database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err = db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Phase 1 Optimization: Configure connection pool for high-performance indexing
	// These settings are optimized for TiDB's distributed nature and bulk operations
	sqlDB.SetMaxIdleConns(200)                 // Increased from 100 for better connection reuse
	sqlDB.SetMaxOpenConns(800)                 // Increased from 400 for higher concurrency
	sqlDB.SetConnMaxLifetime(4 * time.Hour)    // Increased from 2h for stable long-running connections
	sqlDB.SetConnMaxIdleTime(1 * time.Hour)    // Increased from 30m to reduce connection churn

	// Phase 1 Optimization: Enable TiDB-specific optimizations
	if err := enableTiDBOptimizations(db); err != nil {
		log.Printf("Warning: failed to enable some TiDB optimizations: %v", err)
		// Continue anyway as these are optional optimizations
	}

	log.Println("TiDB cluster connection established via HAProxy load balancer with performance optimizations")
	return db, nil
}

// AutoMigrate runs GORM auto-migration for all models
func AutoMigrate(db *gorm.DB) error {
	log.Println("Starting database migrations...")

	// For TiDB, we need to handle charset conflicts more carefully
	if err := handleTiDBMigration(db); err != nil {
		return fmt.Errorf("TiDB migration failed: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// handleTiDBMigration handles TiDB-specific migration issues
func handleTiDBMigration(db *gorm.DB) error {
	// Disable foreign key checks during migration
	if err := db.Exec("SET foreign_key_checks = 0").Error; err != nil {
		log.Printf("Warning: could not disable foreign key checks: %v", err)
	}

	// Check if we need to drop tables due to charset conflicts
	if needsTableDrop(db) {
		log.Println("Detected schema conflicts, dropping tables for clean migration...")
		if err := dropAllTables(db); err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}
	}

	// Core models (order matters due to foreign keys)
	coreModels := []interface{}{
		&models.SlotLeader{},
		&models.Epoch{},
		&models.Block{},
		&models.Tx{},
		&models.StakeAddress{},
		&models.PoolHash{},
		&models.Script{},
		&models.Datum{},
		&models.RedeemerData{},
		&models.TxOut{},
		&models.TxIn{},
		&models.Redeemer{},
	}

	// Asset models
	assetModels := []interface{}{
		&models.MultiAsset{},
		&models.MaTxOut{},
		&models.MaTxMint{},
		&models.CollateralTxIn{},
		&models.ReferenceTxIn{},
		&models.CollateralTxOut{},
		&models.TxMetadata{},
		&models.ExtraKeyWitness{},
		&models.TxCbor{},
	}

	// Staking models
	stakingModels := []interface{}{
		&models.PoolMetadataRef{}, // Must come before PoolUpdate and off-chain pool models
		&models.PoolUpdate{},      // References PoolMetadataRef
		&models.PoolRelay{},
		&models.PoolRetire{},
		&models.PoolOwner{},
		&models.PoolStat{},
		&models.OffChainPoolData{},      // Pool-related off-chain data (references PoolHash, PoolMetadataRef)
		&models.OffChainPoolFetchError{}, // Pool metadata fetch errors (references PoolHash, PoolMetadataRef)
		&models.StakeRegistration{},
		&models.StakeDeregistration{},
		&models.Delegation{},
		&models.Reward{},
		&models.RewardRest{},
		&models.Withdrawal{},
		&models.EpochStake{},
		&models.EpochStakeProgress{},
		&models.DelistedPool{},
		&models.ReservedPoolTicker{},
	}

	// Governance models
	governanceModels := []interface{}{
		&models.CostModel{},
		&models.EpochParam{},
		&models.VotingAnchor{},
		&models.DRepHash{},
		&models.CommitteeHash{},
		&models.ParamProposal{},
		&models.GovActionProposal{},
		&models.VotingProcedure{},
		&models.DelegationVote{},
		&models.CommitteeRegistration{},
		&models.CommitteeDeregistration{},
		&models.TreasuryWithdrawal{},
		&models.CommitteeMember{},
		&models.EpochState{},
		&models.DRepDistr{},
		&models.Committee{},
		&models.Constitution{},
		&models.Treasury{},
		&models.Reserve{},
		&models.PotTransfer{},
	}

	// Off-chain models (vote-related only, pool models moved to staking section)
	offChainModels := []interface{}{
		&models.OffChainVoteData{},
		&models.OffChainVoteGovActionData{},
		&models.OffChainVoteDRepData{},
		&models.OffChainVoteAuthor{},
		&models.OffChainVoteReference{},
		&models.OffChainVoteExternalUpdate{},
		&models.OffChainVoteFetchError{},
	}

	// Utility models
	utilityModels := []interface{}{
		&models.AdaPots{},
		&models.EventInfo{},
	}

	// Migrate in order
	allModels := [][]interface{}{
		coreModels,
		assetModels,
		stakingModels,
		governanceModels,
		offChainModels,
		utilityModels,
	}

	for i, modelGroup := range allModels {
		log.Printf("Migrating model group %d/%d...", i+1, len(allModels))

		if err := db.AutoMigrate(modelGroup...); err != nil {
			return fmt.Errorf("failed to migrate model group %d: %w", i+1, err)
		}
	}

	// Re-enable foreign key checks
	if err := db.Exec("SET foreign_key_checks = 1").Error; err != nil {
		log.Printf("Warning: could not re-enable foreign key checks: %v", err)
	}

	// Apply TiDB-specific optimizations after migration
	if err := applyTiDBOptimizations(db); err != nil {
		log.Printf("Warning: failed to apply TiDB optimizations: %v", err)
		// Don't fail the migration if optimizations fail
	}

	return nil
}

// needsTableDrop checks if we need to drop tables due to schema conflicts
func needsTableDrop(db *gorm.DB) bool {
	// Check if blocks table exists and has charset conflicts
	var result struct {
		Count int
	}

	query := `
		SELECT COUNT(*) as count 
		FROM information_schema.columns 
		WHERE table_schema = DATABASE() 
		AND table_name = 'blocks' 
		AND column_name = 'vrf_key'
		AND character_set_name = 'binary'
	`

	if err := db.Raw(query).Scan(&result).Error; err != nil {
		// If we can't check, assume we don't need to drop
		return false
	}

	return result.Count > 0
}

// dropAllTables drops all tables in the correct order to handle foreign keys
func dropAllTables(db *gorm.DB) error {
	// Tables to drop in reverse dependency order
	tables := []string{
		"off_chain_vote_fetch_errors", "off_chain_vote_external_updates", "off_chain_vote_references",
		"off_chain_vote_authors", "off_chain_vote_d_rep_data", "off_chain_vote_gov_action_data",
		"off_chain_vote_data", "off_chain_pool_fetch_errors", "off_chain_pool_data",
		"pot_transfers", "reserves", "treasuries", "constitutions", "committees",
		"d_rep_distrs", "epoch_states", "committee_members", "treasury_withdrawals",
		"committee_deregistrations", "committee_registrations", "delegation_votes",
		"voting_procedures", "gov_action_proposals", "param_proposals", "committee_hashes",
		"d_rep_hashes", "voting_anchors", "epoch_params", "cost_models",
		"reserved_pool_tickers", "delisted_pools", "epoch_stake_progress", "epoch_stakes",
		"withdrawals", "reward_rests", "rewards", "delegations", "stake_deregistrations",
		"stake_registrations", "pool_stats", "pool_owners", "pool_retires", "pool_relays",
		"pool_updates", "pool_metadata_refs",
		"tx_cbors", "extra_key_witnesses", "tx_metadata", "collateral_tx_outs",
		"reference_tx_ins", "collateral_tx_ins", "ma_tx_mints", "ma_tx_outs", "multi_assets",
		"redeemers", "tx_ins", "tx_outs", "redeemer_data", "datums", "scripts", "pool_hashes",
		"stake_addresses", "txes", "blocks", "epochs", "slot_leaders", "ada_pots", "event_info",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Error; err != nil {
			log.Printf("Warning: failed to drop table %s: %v", table, err)
			// Continue dropping other tables
		}
	}

	log.Println("All tables dropped successfully")
	return nil
}

// applyTiDBOptimizations applies TiDB-specific optimizations
func applyTiDBOptimizations(db *gorm.DB) error {
	log.Println("Applying TiDB-specific optimizations...")

	// Apply sharding for high-volume tables
	shardingQueries := []string{
		"ALTER TABLE blocks SHARD_ROW_ID_BITS = 4",
		"ALTER TABLE txes SHARD_ROW_ID_BITS = 6",
		"ALTER TABLE tx_outs SHARD_ROW_ID_BITS = 8",
		"ALTER TABLE tx_ins SHARD_ROW_ID_BITS = 8",
		"ALTER TABLE voting_procedures SHARD_ROW_ID_BITS = 6",
		"ALTER TABLE ma_tx_outs SHARD_ROW_ID_BITS = 8",
		"ALTER TABLE stake_addresses SHARD_ROW_ID_BITS = 4",
	}

	for _, query := range shardingQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("Warning: failed to apply sharding (%s): %v", query, err)
			// Continue with other optimizations
		}
	}

	// Phase 1 Optimization: Create comprehensive indexes for common queries
	// NOTE: These indexes are now defined in the model struct tags and created by auto-migration.
	// We keep these manual creations as a safety net for existing installations that might
	// not have recreated their tables. The "IF NOT EXISTS" clause makes this safe.
	indexQueries := []string{
		// CRITICAL: Missing hash_raw index was causing 55K+ record table scans
		// Now defined in models/staking.go with gorm:"uniqueIndex:idx_stake_addresses_hash_raw"
		"CREATE INDEX IF NOT EXISTS idx_stake_addresses_hash_raw ON stake_addresses(hash_raw)",

		// Performance indexes for tx_outs (high-volume table)
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_compound ON tx_outs(tx_id, stake_address_id)",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_lookup ON tx_outs(tx_id, `index`)",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_tx_id ON tx_outs(tx_id)", // For batch lookups

		// Pool hash lookup optimization
		"CREATE INDEX IF NOT EXISTS idx_pool_hashes_hash_raw ON pool_hashes(hash_raw)",

		// Multi-asset lookup optimization  
		"CREATE INDEX IF NOT EXISTS idx_multi_assets_policy_name ON multi_assets(policy, name)",

		// Transaction and block relationship indexes
		"CREATE INDEX IF NOT EXISTS idx_txes_block_id ON txes(block_id)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_no ON blocks(slot_no)",

		// Certificate processing indexes
		"CREATE INDEX IF NOT EXISTS idx_delegations_addr_id ON delegations(addr_id)",
		"CREATE INDEX IF NOT EXISTS idx_pool_updates_hash_id ON pool_updates(hash_id)",

		// Governance indexes
		"CREATE INDEX IF NOT EXISTS idx_voting_procedures_tx_id_index ON voting_procedures(tx_id, `index`)",
		"CREATE INDEX IF NOT EXISTS idx_gov_action_proposals_tx_id_index ON gov_action_proposals(tx_id, `index`)",

		// Existing indexes
		"CREATE INDEX IF NOT EXISTS idx_voting_procedure_voter_role_epoch ON voting_procedures (voter_role, tx_id)",
		"CREATE INDEX IF NOT EXISTS idx_tx_out_address_value ON tx_outs (address(20), value)",
		"CREATE INDEX IF NOT EXISTS idx_delegation_epoch ON delegations (active_epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_gov_action_type ON gov_action_proposals (type, tx_id)",
		"CREATE INDEX IF NOT EXISTS idx_epoch_state_epoch ON epoch_states (epoch_no)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("Warning: failed to create index (%s): %v", query, err)
			// Continue with other indexes
		}
	}

	log.Println("TiDB optimizations applied")
	return nil
}

// enableTiDBOptimizations applies TiDB-specific session and global optimizations
func enableTiDBOptimizations(db *gorm.DB) error {
	// Phase 1 Optimization: TiDB-specific optimizations for bulk operations
	optimizations := []string{
		// Increase batch size limits
		"SET SESSION tidb_batch_insert = ON",
		"SET SESSION tidb_batch_delete = ON",
		"SET SESSION tidb_batch_commit = ON",
		
		// Optimize for bulk operations
		"SET SESSION tidb_dml_batch_size = 5000",
		"SET SESSION tidb_max_chunk_size = 1024",
		
		// Disable unnecessary checks during bulk insert
		"SET SESSION tidb_skip_utf8_check = ON",
		"SET SESSION tidb_constraint_check_in_place = OFF",
		
		// Optimize memory usage
		"SET SESSION tidb_mem_quota_query = 8589934592", // 8GB per query
		"SET SESSION tidb_enable_chunk_rpc = ON",
		
		// Enable parallel execution
		"SET SESSION tidb_distsql_scan_concurrency = 15",
		"SET SESSION tidb_executor_concurrency = 5",
		
		// Optimize index usage
		"SET SESSION tidb_opt_insubq_to_join_and_agg = ON",
		"SET SESSION tidb_enable_index_merge = ON",
		
		// Enable async commit and 1PC for better performance
		"SET SESSION tidb_enable_async_commit = ON",
		"SET SESSION tidb_enable_1pc = ON",
	}
	
	for _, opt := range optimizations {
		if err := db.Exec(opt).Error; err != nil {
			// Some settings might not be available in all TiDB versions
			log.Printf("Warning: TiDB optimization failed (%s): %v", opt, err)
			// Continue with other optimizations
		}
	}
	
	return nil
}

// CheckDatabaseConnection verifies the database connection is healthy
func CheckDatabaseConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}
