# Nectar Project Structure

## Directory Layout

```
nectar/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── nectar.toml            # Configuration file
├── README.md              # Project documentation
│
├── auth/                  # Authentication & authorization
│   └── auth.go           # JWT authentication implementation
│
├── config/                # Configuration management
│   ├── config.go         # Configuration structures
│   ├── config_loader.go  # Configuration loading logic
│   ├── init.go          # Configuration initialization
│   └── interactive_init.go # Interactive setup wizard
│
├── constants/             # Application constants
│   └── constants.go      # Shared constants
│
├── dashboard/             # Web dashboard
│   └── web_adapter.go    # Dashboard adapter
│
├── database/              # Database layer
│   ├── gorm_logger.go    # GORM logging adapter
│   ├── migration_helper.go # Migration utilities
│   ├── mysql_config.go   # MySQL configuration
│   ├── optimize_indexes.go # Index optimization
│   ├── retry.go          # Retry logic
│   ├── tidb.go          # TiDB connection management
│   └── unified_indexes.go # Index definitions
│
├── errors/                # Error handling
│   ├── migration.go      # Migration error handling
│   ├── mysql_logger.go   # MySQL error logging
│   ├── stderr_interceptor.go # Stderr capture
│   └── unified_error_system.go # Unified error system
│
├── logger/                # Logging
│   └── logger.go         # Logging utilities
│
├── metadata/              # Metadata handling
│   └── fetcher.go        # Metadata fetching logic
│
├── models/                # Data models
│   ├── assets.go         # Asset-related models
│   ├── checkpoint.go     # Checkpoint model
│   ├── core.go          # Core blockchain models
│   ├── governance.go     # Governance models
│   ├── offchain.go      # Off-chain data models
│   ├── staking.go       # Staking models
│   └── utxo_tracking.go  # UTXO tracking models
│
├── processors/            # Block processing logic
│   ├── ada_pots_calculator.go    # ADA pots calculation
│   ├── asset_processor.go        # Asset processing
│   ├── block_processor.go        # Main block processor
│   ├── byte_lru_cache.go        # Byte-based LRU cache
│   ├── certificate_processor.go  # Certificate processing
│   ├── epoch_params_provider.go  # Epoch parameters
│   ├── era_config.go            # Era configuration
│   ├── error_collector.go       # Error collection
│   ├── generic_lru_cache.go     # Generic LRU cache
│   ├── governance_processor.go   # Governance processing
│   ├── logging_config.go        # Logging configuration
│   ├── metadata_processor.go     # Metadata processing
│   ├── script_processor.go       # Script processing
│   ├── stake_address_cache.go    # Stake address caching
│   └── withdrawal_processor.go   # Withdrawal processing
│
├── sync/                  # Blockchain synchronization
│   └── sync.go           # Sync logic
│
├── terminal/              # Terminal UI
│   └── progress.go       # Progress display
│
├── tests/                 # Test files
│   └── unit/             # Unit tests
│       ├── processors/   # Processor tests
│       │   ├── block_processor_test.go
│       │   ├── byte_lru_cache_test.go
│       │   ├── certificate_processor_test.go
│       │   └── transaction_processor_test.go
│       └── database/     # Database tests
│           └── retry_test.go
│
└── web/                   # Web server
    ├── handlers.go       # HTTP handlers
    ├── server.go         # Web server setup
    └── templates/        # HTML templates
        ├── index.html
        └── partials/
            └── ...
```

## Package Organization

### Core Packages
- **main**: Application entry point and initialization
- **config**: Configuration management and validation
- **database**: Database connections and query optimization
- **models**: Data structures representing blockchain entities

### Processing Packages
- **processors**: All blockchain data processing logic
  - Block processing pipeline
  - Transaction and UTXO handling
  - Certificate and staking operations
  - Governance actions
  - Asset and metadata processing

### Infrastructure Packages
- **auth**: JWT-based authentication
- **errors**: Unified error handling and logging
- **logger**: Structured logging
- **terminal**: CLI progress display

### Web Packages
- **web**: HTTP server and API endpoints
- **dashboard**: Web dashboard adapter

## Key Design Principles

1. **Separation of Concerns**: Each package has a single, well-defined responsibility
2. **Testability**: Test files are separated from implementation
3. **Modularity**: Processors can be easily extended or modified
4. **Performance**: Caching and optimization built into critical paths
5. **Error Handling**: Unified error system for consistent error management

## Adding New Features

When adding new features:
1. Place business logic in appropriate processor
2. Add data models to `models/` package
3. Write tests in `tests/unit/` directory
4. Update this documentation