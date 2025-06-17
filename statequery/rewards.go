// Copyright 2025 The Nectar Authors
// Rewards calculation and tracking

package statequery

import (
	"fmt"
	"log"
	"nectar/models"
	"os"
	"strconv"

	"gorm.io/gorm"
)

// RewardCalculator handles reward calculations
type RewardCalculator struct {
	db           *gorm.DB
	service      *Service
	lastEpoch    uint32
	rewardParams RewardParameters
}

// RewardParameters holds parameters for reward calculation
type RewardParameters struct {
	TreasuryTax       float64 // τ (tau) - treasury tax
	MonetaryExpansion float64 // ρ (rho) - monetary expansion
	OptimalPoolCount  uint32  // k - optimal number of pools
	InfluenceFactor   float64 // a0 - pool influence factor
	TotalSupply       uint64  // Total ADA supply in lovelace
}

// DefaultRewardParameters returns mainnet reward parameters
// These can be overridden via environment variables
func DefaultRewardParameters() RewardParameters {
	params := RewardParameters{
		TreasuryTax:       0.20,                      // 20% to treasury
		MonetaryExpansion: 0.003,                     // 0.3% per epoch
		OptimalPoolCount:  500,                       // k=500
		InfluenceFactor:   0.3,                       // a0=0.3
		TotalSupply:       45_000_000_000_000_000,    // 45B ADA
	}
	
	// Allow environment overrides for testing or different networks
	if tax := os.Getenv("CARDANO_TREASURY_TAX"); tax != "" {
		if val, err := strconv.ParseFloat(tax, 64); err == nil {
			params.TreasuryTax = val
		}
	}
	
	if expansion := os.Getenv("CARDANO_MONETARY_EXPANSION"); expansion != "" {
		if val, err := strconv.ParseFloat(expansion, 64); err == nil {
			params.MonetaryExpansion = val
		}
	}
	
	if poolCount := os.Getenv("CARDANO_OPTIMAL_POOL_COUNT"); poolCount != "" {
		if val, err := strconv.ParseUint(poolCount, 10, 32); err == nil {
			params.OptimalPoolCount = uint32(val)
		}
	}
	
	return params
}

// NewRewardCalculator creates a new reward calculator
func NewRewardCalculator(db *gorm.DB, service *Service) *RewardCalculator {
	return &RewardCalculator{
		db:           db,
		service:      service,
		rewardParams: DefaultRewardParameters(),
	}
}

// CalculateEpochRewards calculates rewards for a completed epoch
func (rc *RewardCalculator) CalculateEpochRewards(epochNo uint32) error {
	if epochNo < 211 { // Rewards start in epoch 211
		log.Printf("No rewards for epoch %d (rewards start in epoch 211)", epochNo)
		return nil
	}

	log.Printf("Calculating rewards for epoch %d", epochNo)

	// Get epoch data
	var epoch models.Epoch
	if err := rc.db.Where("no = ?", epochNo-2).First(&epoch).Error; err != nil {
		return fmt.Errorf("failed to get epoch %d: %w", epochNo-2, err)
	}

	// Calculate total rewards pot for the epoch
	totalRewardPot := rc.calculateRewardPot(epochNo)
	treasuryReward := uint64(float64(totalRewardPot) * rc.rewardParams.TreasuryTax)
	poolRewards := totalRewardPot - treasuryReward

	log.Printf("Total reward pot: %d, Treasury: %d, Pools: %d", 
		totalRewardPot, treasuryReward, poolRewards)

	// Get active pools and their performance
	pools, err := rc.getActivePoolsForEpoch(epochNo - 2)
	if err != nil {
		return fmt.Errorf("failed to get active pools: %w", err)
	}

	// Calculate rewards for each pool
	tx := rc.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	rewardCount := 0
	for _, pool := range pools {
		// Calculate pool rewards
		poolReward, delegatorRewards := rc.calculatePoolRewards(pool, poolRewards, epochNo-2)
		
		// Store pool leader reward
		if poolReward > 0 && len(pool.RewardAddrHash) > 0 {
			// Verify stake address exists before creating reward
			var exists bool
			if err := tx.Model(&models.StakeAddress{}).Select("count(*) > 0").Where("hash_raw = ?", pool.RewardAddrHash).Find(&exists).Error; err != nil || !exists {
				log.Printf("[WARNING] Skipping leader reward - stake address doesn't exist")
				continue
			}
			
			leaderReward := &models.Reward{
				AddrHash:       pool.RewardAddrHash,
				Type:           "leader",
				Amount:         int64(poolReward),
				EarnedEpoch:    epochNo - 2,
				SpendableEpoch: epochNo + 1,
				PoolHash:       pool.PoolHash,
			}
			if err := tx.Create(leaderReward).Error; err != nil {
				log.Printf("[WARNING] Failed to create leader reward: %v", err)
				continue
			}
			rewardCount++
		}

		// Store delegator rewards
		for addrHashStr, amount := range delegatorRewards {
			if amount > 0 && len(addrHashStr) > 0 {
				// Convert string back to bytes
				addrHash := []byte(addrHashStr)
				
				// Verify stake address exists before creating reward
				var exists bool
				if err := tx.Model(&models.StakeAddress{}).Select("count(*) > 0").Where("hash_raw = ?", addrHash).Find(&exists).Error; err != nil || !exists {
					log.Printf("[WARNING] Skipping delegator reward - stake address doesn't exist")
					continue
				}
				
				delegatorReward := &models.Reward{
					AddrHash:       addrHash,
					Type:           "member",
					Amount:         int64(amount),
					EarnedEpoch:    epochNo - 2,
					SpendableEpoch: epochNo + 1,
					PoolHash:       pool.PoolHash,
				}
				if err := tx.Create(delegatorReward).Error; err != nil {
					log.Printf("[WARNING] Failed to create delegator reward: %v", err)
					continue
				}
				rewardCount++
			}
		}
	}

	// Store unclaimed rewards (reward_rest)
	// These would be rewards that couldn't be distributed
	if treasuryReward > 0 {
		// Find or create system treasury address
		var treasuryAddr models.StakeAddress
		if err := tx.Where("view = ?", "treasury_system").FirstOrCreate(&treasuryAddr, models.StakeAddress{
			HashRaw: []byte("treasury_system_addr"),
			View:    "treasury_system",
		}).Error; err != nil {
			log.Printf("[WARNING] Failed to create treasury address: %v", err)
		} else {
			treasuryRest := &models.RewardRest{
				AddrHash:       treasuryAddr.HashRaw,
				Type:           "treasury",
				Amount:         int64(treasuryReward),
				EarnedEpoch:    epochNo - 2,
				SpendableEpoch: epochNo + 1,
			}
			if err := tx.Create(treasuryRest).Error; err != nil {
				log.Printf("[WARNING] Failed to create treasury reward_rest: %v", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit rewards: %w", err)
	}

	log.Printf("[OK] Created %d rewards for epoch %d", rewardCount, epochNo-2)
	return nil
}

// calculateRewardPot calculates the total reward pot for an epoch
func (rc *RewardCalculator) calculateRewardPot(epochNo uint32) uint64 {
	// Simplified calculation: ρ * (totalSupply - circulatingSupply)
	// In reality, this depends on reserves
	
	// Estimate reserves (decreases over time)
	reserveFraction := 1.0 - (float64(epochNo) * 0.002) // Rough approximation
	if reserveFraction < 0.3 {
		reserveFraction = 0.3 // Minimum reserves
	}
	
	reserves := uint64(float64(rc.rewardParams.TotalSupply) * reserveFraction)
	rewardPot := uint64(float64(reserves) * rc.rewardParams.MonetaryExpansion)
	
	return rewardPot
}

// PoolPerformance holds pool performance data
type PoolPerformance struct {
	PoolHash       []byte
	Pledge         uint64
	Stake          uint64
	BlocksMade     uint32
	RewardAddrHash []byte
	Margin         float64
	Cost           uint64
}

// getActivePoolsForEpoch gets pools that were active in the epoch
func (rc *RewardCalculator) getActivePoolsForEpoch(epochNo uint32) ([]PoolPerformance, error) {
	var pools []PoolPerformance

	query := `
		SELECT DISTINCT 
			ph.hash_raw as pool_hash,
			pu.pledge,
			COALESCE(ps.delegated_stake, 0) as stake,
			COALESCE(ps.block_cnt, 0) as blocks_made,
			pu.reward_addr_hash,
			pu.margin,
			pu.fixed_cost as cost
		FROM pool_hashes ph
		JOIN pool_updates pu ON ph.hash_raw = pu.pool_hash
		LEFT JOIN pool_stats ps ON ph.hash_raw = ps.pool_hash AND ps.epoch_no = ?
		WHERE pu.active_epoch_no <= ?
		AND NOT EXISTS (
			SELECT 1 FROM pool_retires pr 
			WHERE pr.pool_hash = ph.hash_raw 
			AND pr.retiring_epoch <= ?
		)
	`

	if err := rc.db.Raw(query, epochNo, epochNo, epochNo).Scan(&pools).Error; err != nil {
		return nil, err
	}

	return pools, nil
}

// calculatePoolRewards calculates rewards for a pool and its delegators
func (rc *RewardCalculator) calculatePoolRewards(pool PoolPerformance, totalPoolRewards uint64, epochNo uint32) (uint64, map[string]uint64) {
	// Simplified reward calculation
	// In reality, this is much more complex and involves:
	// - Pool saturation
	// - Pool performance (blocks made vs expected)
	// - Pledge influence
	// - Pool parameters (margin, cost)

	// Calculate pool's share of total rewards based on stake
	var totalStake uint64
	rc.db.Model(&models.PoolStat{}).
		Where("epoch_no = ?", epochNo).
		Select("SUM(delegated_stake)").
		Scan(&totalStake)

	if totalStake == 0 || pool.Stake == 0 {
		return 0, nil
	}

	// Pool's base rewards
	poolShare := float64(pool.Stake) / float64(totalStake)
	poolTotalReward := uint64(float64(totalPoolRewards) * poolShare)

	// Apply pool performance factor (simplified)
	performanceFactor := 1.0
	if pool.BlocksMade == 0 {
		performanceFactor = 0.0 // No blocks = no rewards
	}

	poolTotalReward = uint64(float64(poolTotalReward) * performanceFactor)

	// Calculate pool operator rewards (cost + margin)
	operatorReward := pool.Cost
	if poolTotalReward > pool.Cost {
		marginReward := uint64(float64(poolTotalReward-pool.Cost) * pool.Margin)
		operatorReward += marginReward
	}

	// Remaining rewards go to delegators
	delegatorRewards := make(map[string]uint64)
	if poolTotalReward > operatorReward {
		remainingRewards := poolTotalReward - operatorReward
		
		// Get delegators for this pool
		var delegations []struct {
			AddrHash []byte
			Amount   uint64
		}
		
		// Query delegations active during this epoch
		query := `
			SELECT 
				d.addr_hash,
				COALESCE(SUM(txo.value), 0) as amount
			FROM delegations d
			JOIN tx_outs txo ON txo.stake_address_hash = d.addr_hash
			WHERE d.pool_hash = ?
			AND d.active_epoch_no <= ?
			AND NOT EXISTS (
				SELECT 1 FROM stake_deregistrations sd
				WHERE sd.addr_hash = d.addr_hash
				AND sd.tx_hash IN (
					SELECT hash FROM txes t
					JOIN blocks b ON t.block_hash = b.hash
					WHERE b.epoch_no <= ?
				)
			)
			GROUP BY d.addr_hash
		`
		
		if err := rc.db.Raw(query, pool.PoolHash, epochNo, epochNo).Scan(&delegations).Error; err == nil {
			// Distribute rewards proportionally
			var totalDelegated uint64
			for _, d := range delegations {
				totalDelegated += d.Amount
			}
			
			if totalDelegated > 0 {
				for _, d := range delegations {
					share := float64(d.Amount) / float64(totalDelegated)
					reward := uint64(float64(remainingRewards) * share)
					if reward > 0 {
						// Use hex string of address hash as map key
						delegatorRewards[string(d.AddrHash)] = reward
					}
				}
			}
		}
	}

	return operatorReward, delegatorRewards
}

// ProcessRefunds processes deposit refunds from deregistrations
func (rc *RewardCalculator) ProcessRefunds(epochNo uint32) error {
	log.Printf("Processing refunds for epoch %d", epochNo)

	// Find stake deregistrations that should be refunded in this epoch
	// Note: We need to join with blocks to get epoch info
	var deregistrations []struct {
		AddrHash []byte
		Deposit  int64
	}
	
	query := `
		SELECT sd.addr_hash, 2000000 as deposit
		FROM stake_deregistrations sd
		JOIN txes t ON sd.tx_hash = t.hash
		JOIN blocks b ON t.block_hash = b.hash
		WHERE b.epoch_no = ?
		GROUP BY sd.addr_hash
	`
	
	err := rc.db.Raw(query, epochNo-2).Scan(&deregistrations).Error
	if err != nil {
		return fmt.Errorf("failed to get deregistrations: %w", err)
	}

	tx := rc.db.Begin()
	refundCount := 0

	for _, dereg := range deregistrations {
		if dereg.Deposit > 0 && len(dereg.AddrHash) > 0 {
			// Verify stake address exists before creating refund
			var exists bool
			if err := tx.Model(&models.StakeAddress{}).Select("count(*) > 0").Where("hash_raw = ?", dereg.AddrHash).Find(&exists).Error; err != nil || !exists {
				log.Printf("[WARNING] Skipping refund - stake address doesn't exist")
				continue
			}
			
			refund := &models.Reward{
				AddrHash:       dereg.AddrHash,
				Type:           "refund",
				Amount:         dereg.Deposit,
				EarnedEpoch:    epochNo - 2,
				SpendableEpoch: epochNo,
				PoolHash:       nil, // No pool for refunds
			}
			
			if err := tx.Create(refund).Error; err != nil {
				log.Printf("[WARNING] Failed to create refund: %v", err)
				continue
			}
			refundCount++
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit refunds: %w", err)
	}

	log.Printf("[OK] Created %d refunds for epoch %d", refundCount, epochNo)
	return nil
}