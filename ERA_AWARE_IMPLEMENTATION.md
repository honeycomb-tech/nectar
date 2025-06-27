# Era-Aware Dynamic Configuration Implementation

## Overview
Implemented a dynamic configuration system that automatically adjusts performance parameters based on the current Cardano era. This prevents resource exhaustion and queue overflow errors when transitioning from simple Byron blocks to complex Shelley+ blocks.

## Key Components

### 1. Era Configuration (`processors/era_config.go`)
Defines performance parameters for each era:
- **Byron (epochs 0-207)**: Large batches, many workers
- **Shelley (epochs 208-235)**: Reduced batches for staking complexity
- **Allegra/Mary (epochs 236-289)**: Further reduction for native tokens
- **Alonzo (epochs 290-364)**: Smaller batches for smart contracts
- **Babbage (epochs 365+)**: Minimal batches for heavy DeFi

### 2. Block Processor Updates
- Added `currentEraConfig` and `currentEpoch` tracking
- Automatically detects epoch changes and updates configuration
- Uses dynamic batch sizes from era config instead of constants
- Logs era transitions for visibility

### 3. Configuration Values by Era

| Era | Workers | TxBatch | TxOutBatch | FetchRange | Queue |
|-----|---------|---------|------------|------------|-------|
| Byron | 16 | 1000 | 2000 | 5000 | 100K |
| Shelley | 12 | 500 | 1000 | 2000 | 50K |
| Allegra/Mary | 10 | 400 | 800 | 1500 | 30K |
| Alonzo | 8 | 300 | 500 | 1000 | 20K |
| Babbage | 6 | 200 | 300 | 500 | 10K |

## Benefits
1. **Automatic optimization**: No manual intervention needed
2. **Prevents resource exhaustion**: Scales down as complexity increases
3. **Maintains performance**: Uses optimal settings for each era
4. **Visible transitions**: Logs when entering new eras

## Usage
The system automatically detects the current epoch and applies appropriate settings. When Nectar processes blocks, it will:
1. Check if epoch has changed
2. Update configuration if needed
3. Log the transition
4. Continue processing with new settings

## Example Log Output
```
[ERA TRANSITION] Entering Shelley era at epoch 208
[ERA CONFIG] Workers: 12, TxBatch: 500, TxOutBatch: 1000, FetchRange: 2000
```

This ensures smooth operation throughout the entire blockchain history without manual configuration changes.