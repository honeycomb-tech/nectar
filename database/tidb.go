package database

import (
	"fmt"
	"log"
	"nectar/errors"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	// DefaultTiDBDSN is used only if TIDB_DSN environment variable is not set
	// In production, always set TIDB_DSN with proper credentials
	DefaultTiDBDSN = "root:@tcp(127.0.0.1:4000)/nectar?charset=utf8mb4&parseTime=True&loc=Local&timeout=300s&readTimeout=300s&writeTimeout=300s&maxAllowedPacket=67108864&autocommit=true&tx_isolation='READ-COMMITTED'"
	// Deprecated: Use NECTAR_DSN environment variable instead
	NectarDBDSN    = "root:@tcp(127.0.0.1:4000)/nectar?charset=utf8mb4&parseTime=True&loc=Local&timeout=300s&readTimeout=300s&writeTimeout=300s&maxAllowedPacket=67108864&autocommit=true&tx_isolation='READ-COMMITTED'"
)

// InitTiDB initializes the TiDB connection with optimizations
func InitTiDB() (*gorm.DB, error) {
	// Get DSN from environment or use default
	baseDSN := os.Getenv("TIDB_DSN")
	if baseDSN == "" {
		// Log warning if using default DSN
		log.Println("WARNING: TIDB_DSN not set, using default DSN without password. Set TIDB_DSN in production!")
		baseDSN = DefaultTiDBDSN
	}

	// First, connect without specifying a database
	baseDB, err := gorm.Open(mysql.Open(baseDSN), &gorm.Config{
		Logger: errors.NewGormLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TiDB cluster: %w", err)
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

	// Now connect to the nectar database
	nectarDSN := os.Getenv("NECTAR_DSN")
	if nectarDSN == "" {
		// Try TIDB_DSN as fallback
		nectarDSN = os.Getenv("TIDB_DSN")
		if nectarDSN == "" {
			log.Println("WARNING: NECTAR_DSN not set, using default DSN without password. Set NECTAR_DSN in production!")
			nectarDSN = NectarDBDSN
		}
	}

	db, err := gorm.Open(mysql.Open(nectarDSN), &gorm.Config{
		Logger:                                   errors.NewGormLogger(),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nectar database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err = db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool for high-performance
	sqlDB.SetMaxIdleConns(400)                 // Increased for better reuse
	sqlDB.SetMaxOpenConns(1000)                // More concurrent connections
	sqlDB.SetConnMaxLifetime(4 * time.Hour)    
	sqlDB.SetConnMaxIdleTime(1 * time.Hour)    

	// Enable TiDB-specific optimizations
	if err := enableTiDBOptimizations(db); err != nil {
		log.Printf("Warning: failed to enable some TiDB optimizations: %v", err)
	}

	log.Println("TiDB connection established with performance optimizations")
	return db, nil
}

// AutoMigrate runs GORM auto-migration for all models
func AutoMigrate(db *gorm.DB) error {
	log.Println("Starting database migrations...")

	// Disable foreign key checks during migration
	if err := db.Exec("SET foreign_key_checks = 0").Error; err != nil {
		log.Printf("Warning: could not disable foreign key checks: %v", err)
	}

	// Create all tables without foreign key constraints first
	if err := createAllTablesWithoutForeignKeys(db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Re-enable foreign key checks
	if err := db.Exec("SET foreign_key_checks = 1").Error; err != nil {
		log.Printf("Warning: could not re-enable foreign key checks: %v", err)
	}

	// Apply TiDB-specific optimizations after migration
	if err := applyTiDBOptimizations(db); err != nil {
		log.Printf("Warning: failed to apply TiDB optimizations: %v", err)
	}

	// Create performance indexes
	if err := createPerformanceIndexes(db); err != nil {
		log.Printf("Warning: failed to create performance indexes: %v", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// applyTiDBOptimizations applies TiDB-specific optimizations
func applyTiDBOptimizations(db *gorm.DB) error {
	log.Println("Applying TiDB-specific optimizations...")

	// Create indexes for better query performance
	indexQueries := []string{
		// Hash-based lookups are already primary keys, but we need some additional indexes
		"CREATE INDEX IF NOT EXISTS idx_blocks_epoch_no ON blocks(epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_time ON blocks(time)",
		"CREATE INDEX IF NOT EXISTS idx_txes_block_hash ON txes(block_hash)",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_stake_address ON tx_outs(stake_address_hash)",
		"CREATE INDEX IF NOT EXISTS idx_tx_ins_spent ON tx_ins(tx_out_hash, tx_out_index)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_epoch ON delegations(active_epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_rewards_epoch ON rewards(earned_epoch)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("Warning: failed to create index (%s): %v", query, err)
		}
	}

	log.Println("TiDB optimizations applied")
	return nil
}

// enableTiDBOptimizations applies TiDB-specific session optimizations
func enableTiDBOptimizations(db *gorm.DB) error {
	optimizations := []string{
		// Batch operation optimizations
		"SET SESSION tidb_batch_insert = ON",
		"SET SESSION tidb_batch_delete = ON",
		"SET SESSION tidb_batch_commit = ON",
		"SET SESSION tidb_dml_batch_size = 20000",
		"SET SESSION tidb_max_chunk_size = 1024",
		
		// Skip unnecessary checks
		"SET SESSION tidb_skip_utf8_check = ON",
		"SET SESSION tidb_constraint_check_in_place = OFF",
		
		// Memory optimizations  
		"SET SESSION tidb_mem_quota_query = 17179869184",  // 16GB per query
		"SET SESSION tidb_enable_chunk_rpc = ON",
		
		// Parallel execution
		"SET SESSION tidb_distsql_scan_concurrency = 30",
		"SET SESSION tidb_executor_concurrency = 10",
		
		// Index optimizations
		"SET SESSION tidb_opt_insubq_to_join_and_agg = ON",
		"SET SESSION tidb_enable_index_merge = ON",
		
		// Transaction optimizations
		"SET SESSION tidb_enable_async_commit = ON",
		"SET SESSION tidb_enable_1pc = ON",
		
		// Additional optimizations for high-throughput processing
		"SET SESSION tidb_enable_vectorized_expression = ON",
		"SET SESSION tidb_projection_concurrency = 8",
		"SET SESSION tidb_hash_join_concurrency = 8",
		"SET SESSION tidb_enable_parallel_apply = ON",
		// Note: tidb_enable_batch_dml is a GLOBAL variable, skip it for session
	}
	
	for _, opt := range optimizations {
		if err := db.Exec(opt).Error; err != nil {
			log.Printf("Warning: TiDB optimization failed (%s): %v", opt, err)
		}
	}
	
	return nil
}

// createPerformanceIndexes creates additional indexes for performance optimization
func createPerformanceIndexes(db *gorm.DB) error {
	log.Println("Creating performance indexes...")
	
	performanceIndexes := []string{
		// Index for fast lookup of latest blocks (fixes slow query)
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_no_desc ON blocks(slot_no DESC)",
		"CREATE INDEX IF NOT EXISTS idx_blocks_slot_hash ON blocks(slot_no DESC, hash)",
		
		// Additional performance indexes for common queries
		"CREATE INDEX IF NOT EXISTS idx_blocks_block_no_desc ON blocks(block_no DESC)",
		"CREATE INDEX IF NOT EXISTS idx_txes_block_index ON txes(block_hash, block_index)",
		"CREATE INDEX IF NOT EXISTS idx_tx_outs_value ON tx_outs(value)",
		"CREATE INDEX IF NOT EXISTS idx_delegations_addr_epoch ON delegations(addr_hash, active_epoch_no)",
		"CREATE INDEX IF NOT EXISTS idx_pool_stats_epoch_hash ON pool_stats(epoch_no, pool_hash)",
	}
	
	for _, index := range performanceIndexes {
		log.Printf("Creating index: %s", index)
		if err := db.Exec(index).Error; err != nil {
			log.Printf("Warning: failed to create performance index (%s): %v", index, err)
			// Continue with other indexes even if one fails
		}
	}
	
	log.Println("Performance indexes created")
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