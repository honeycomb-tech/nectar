# Processors Package

This package contains all the blockchain data processing logic for Nectar.

## File Organization

### Core Processors
- `block_processor.go` - Main block processing pipeline
- `asset_processor.go` - Native asset and token processing
- `metadata_processor.go` - Transaction metadata handling
- `script_processor.go` - Plutus script processing

### Staking & Certificates
- `certificate_processor.go` - Stake pool and delegation certificates
- `stake_address_cache.go` - Stake address caching layer
- `withdrawal_processor.go` - Reward withdrawals
- `ada_pots_calculator.go` - ADA distribution calculations
- `epoch_params_provider.go` - Epoch parameter management

### Governance
- `governance_processor.go` - Conway era governance actions

### Infrastructure
- `byte_lru_cache.go` - Byte-based LRU cache (for slot leaders)
- `generic_lru_cache.go` - Generic LRU cache (for scripts/datums)
- `error_collector.go` - Centralized error collection
- `era_config.go` - Era-specific configurations
- `logging_config.go` - Logging configuration

## Adding New Processors

When adding a new processor:
1. Create a struct that embeds necessary dependencies
2. Implement a `New*Processor()` constructor
3. Add processing methods following existing patterns
4. Update `block_processor.go` if needed
5. Add tests in `tests/unit/processors/`

## Testing

Test files are located in `tests/unit/processors/` to keep the main package clean.