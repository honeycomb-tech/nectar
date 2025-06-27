# Shelley+ Era Optimization Guide

## The Problem
When Nectar reaches Shelley era (epoch 208+), performance degrades because:
1. Blocks become much more complex (staking, delegations, certificates)
2. Transaction sizes increase dramatically
3. TiKV struggles with the increased write load
4. Workers can't keep up, causing queue overflow

## Quick Fix (Applied)
Reduced configuration in nectar.toml:
- Workers: 16 → 8 (less concurrent load on TiKV)
- Fetch range: 2000 → 500 blocks
- Queue size: 100K → 10K blocks

## Batch Size Recommendations by Era

### Byron (Epochs 0-207)
- Simple transactions, fast processing
- Can use larger batches (1000-5000)

### Shelley (Epochs 208-235)
- Staking and delegation start
- Reduce batches to 500-1000

### Allegra/Mary (Epochs 236-289)
- Native tokens introduced
- Keep batches at 500

### Alonzo (Epochs 290-364)
- Smart contracts begin
- Reduce to 200-500

### Babbage (Epochs 365+)
- Heavy smart contract usage
- Use smallest batches (100-200)

## Monitoring Commands
```bash
# Check TiKV CPU usage
ps aux | grep tikv | grep -v grep

# Check queue status
tail -f nectar.log | grep "queue full"

# Monitor sync speed
watch -n 5 'mysql -h127.0.0.1 -P4100 -uroot -pPASSWORD -e "SELECT MAX(block_no), MAX(epoch_no) FROM nectar.blocks"'
```

## Dynamic Adjustment
Consider implementing era-aware batch sizing:
```go
func getBatchSize(epochNo int) int {
    switch {
    case epochNo < 208:  // Byron
        return 1000
    case epochNo < 290:  // Shelley/Allegra/Mary
        return 500
    case epochNo < 365:  // Alonzo
        return 300
    default:             // Babbage+
        return 200
    }
}
```