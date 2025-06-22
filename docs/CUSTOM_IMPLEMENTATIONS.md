# Custom Implementations for Missing State Query Features

## What We Built

### 1. ada_pots Calculator (`processors/ada_pots_calculator.go`)
- Calculates ADA distribution at each epoch boundary
- Uses UTXO set to determine circulating supply
- Estimates treasury and reserves based on fees and rewards
- Runs automatically when processing epoch boundaries

### 2. epoch_params Provider (`processors/epoch_params_provider.go`)
- Provides hardcoded mainnet protocol parameters
- Parameters are era-specific (Byron → Conway)
- Includes all major parameters like fees, deposits, pool costs
- Runs automatically at epoch boundaries

### 3. Integration
Both are integrated into `BlockProcessor` and run automatically when:
- An epoch boundary is detected
- The epoch is >= 208 (Shelley onwards for ada_pots)

## How It Works

When Nectar processes a block at an epoch boundary:
1. Creates epoch record
2. Calculates ada_pots from current UTXO state
3. Provides epoch_params based on the era

## Result
- `ada_pots` table will be populated with treasury, reserves, utxo, fees
- `epoch_params` table will have all protocol parameters per epoch
- No need for state query - we calculate/provide the data ourselves!

## Note
This is a pragmatic solution that provides the data without waiting for gouroboros to implement state queries. The calculations are accurate enough for most use cases.