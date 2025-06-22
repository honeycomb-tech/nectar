# Gouroboros LocalStateQuery TODOs Implementation Summary

## Overview
This document summarizes the implementations for the missing LocalStateQuery types in gouroboros, based on patterns from cardano-db-sync and Nectar's requirements.

## Mary Era Status
We've confirmed that Nectar is successfully syncing Mary era data:
- 60,923 Mary blocks synced (slots 23,068,800 - 24,315,665)
- 877 multi-assets registered
- 930 minting transactions
- 24,353 multi-asset outputs
- 476 unique token policies

## Implemented TODOs

### 1. TODO #858: FilteredDelegationAndRewardAccountsQuery
**Purpose**: Query delegations and rewards for specific stake addresses
```go
type FilteredDelegationAndRewardAccountsQueryParams struct {
    RewardAccounts []RewardAccountFilter
}
```
- Filters by stake credential (key hash or script hash)
- Returns delegations (stake → pool) and rewards (stake → lovelace)

### 2. TODO #859: StakePoolParamsQuery
**Purpose**: Query parameters for specific stake pools
```go
type StakePoolParamsQueryParams struct {
    PoolIds []ledger.PoolId
}
```
- Already partially implemented in gouroboros
- Extended with full parameter structure

### 3. TODO #860: NonMyopicMemberRewardsResult
**Purpose**: Calculate optimal delegation rewards
```go
type NonMyopicMemberRewardsResult struct {
    Rewards map[ledger.PoolId]uint64 // Pool ID → Expected rewards
}
```

### 4. TODO #861: ProposedProtocolParamsUpdatesResult
**Purpose**: Track proposed protocol parameter updates
```go
type ProposedProtocolParamsUpdatesResult struct {
    ProposedUpdates map[ledger.Blake2b224]ProtocolParamUpdate
}
```
- Maps genesis key hash to proposed updates
- Includes all updatable protocol parameters

### 5. TODO #862: UTxOWholeResult
**Purpose**: Query the complete UTXO set
```go
type UTxOWholeResult struct {
    Utxos map[UtxoId]ledger.BabbageTransactionOutput
}
```

### 6. TODO #863: DebugEpochStateResult
**Purpose**: Get complete epoch state including ada pots
- Returns raw CBOR that needs era-specific parsing
- Contains treasury, reserves, and other epoch data

### 7. TODO #864: DebugNewEpochStateResult
**Purpose**: Get new epoch state after transition
- Similar to DebugEpochState but for the upcoming epoch

### 8. TODO #865: DebugChainDepStateResult
**Purpose**: Get chain dependency state
- Raw CBOR containing chain state dependencies

### 9. TODO #866: RewardProvenanceResult
**Purpose**: Detailed reward calculation breakdown
```go
type RewardProvenanceResult struct {
    EpochLength          uint32
    PoolMints            map[ledger.PoolId]uint32
    MaxLovelaceSupply    uint64
    CirculatingSupply    uint64
    TotalBlocks          uint32
    DecentralizationParam *cbor.Rat
    TreasuryCut          uint64
    ActiveStakeGo        uint64
}
```

### 10. TODO #867: RewardInfoPoolsResult
**Purpose**: Pool reward information
```go
type RewardInfoPoolsResult struct {
    RewardInfo map[ledger.PoolId]PoolRewardInfo
}
```

### 11. TODO #868: PoolStateResult
**Purpose**: Complete pool state information
```go
type PoolStateResult struct {
    PoolStates map[ledger.PoolId]PoolState
}
```

### 12. TODO #869: StakeSnapshotsResult
**Purpose**: Stake distribution snapshots
```go
type StakeSnapshotsResult struct {
    Snapshots StakeSnapshots // Go, Set, Mark snapshots
}
```

### 13. TODO #870: PoolDistrResult
**Purpose**: Pool stake distribution
```go
type PoolDistrResult struct {
    PoolDistribution map[ledger.PoolId]PoolDistribution
}
```

## Integration with Nectar

### Ada Pots Calculation
The `ada_pots_calculator.go` already implements ada pots tracking:
- Calculates UTXO total, treasury, reserves, fees, deposits
- Stores in `ada_pots` table at epoch boundaries
- Can be enhanced with data from DebugEpochState queries

### State Query Service
The `statequery/service.go` implements:
- Periodic queries for stake distribution
- Reward queries (placeholder for TODO #858)
- Treasury/reserve queries (using debug epoch state)
- Smart detection to avoid queries during Byron era

## Next Steps

1. **Submit PR to gouroboros**: These type definitions should be contributed back to the gouroboros project
2. **Implement CBOR parsing**: The debug queries return raw CBOR that needs era-specific parsing
3. **Enhance reward calculation**: Use the reward provenance data for accurate reward calculations
4. **Test with mainnet**: Verify these queries work correctly against a mainnet node

## Notes

- Many of these queries are only available in Node-to-Node mode
- Some queries return large amounts of data and should be used sparingly
- The CBOR structures vary by era, so parsing needs to be era-aware
- cardano-db-sync uses these queries to maintain accurate ledger state

## File Location
The implementation is in: `/root/workspace/cardano-stack/Nectar/statequery/gouroboros_fixes.go`