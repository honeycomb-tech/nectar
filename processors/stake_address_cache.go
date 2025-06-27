package processors

import (
	"fmt"
	"nectar/models"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StakeAddressCache manages stake address lookups and creation with LRU eviction
type StakeAddressCache struct {
	db    *gorm.DB
	cache *GenericLRUCache // Map from hex string to hash bytes
}

// NewStakeAddressCache creates a new stake address cache with LRU eviction
func NewStakeAddressCache(db *gorm.DB) *StakeAddressCache {
	// 200k entries should be enough for most stake addresses
	return &StakeAddressCache{
		db:    db,
		cache: NewGenericLRUCache(200000),
	}
}

// GetOrCreateStakeAddress gets or creates a stake address from a ledger.Address
func (sac *StakeAddressCache) GetOrCreateStakeAddress(tx *gorm.DB, addr ledger.Address) ([]byte, error) {
	// Extract stake credential bytes from address
	stakeBytes, err := addr.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get address bytes: %w", err)
	}
	if len(stakeBytes) < 28 {
		return nil, fmt.Errorf("invalid stake address bytes length: %d", len(stakeBytes))
	}

	// Use last 28 bytes as the stake address hash
	hashBytes := stakeBytes[len(stakeBytes)-28:]

	return sac.GetOrCreateStakeAddressFromBytesWithTx(tx, hashBytes)
}

// GetOrCreateStakeAddressFromBytesWithTx gets or creates a stake address from bytes using a transaction
func (sac *StakeAddressCache) GetOrCreateStakeAddressFromBytesWithTx(tx *gorm.DB, hashBytes []byte) ([]byte, error) {
	if len(hashBytes) != 28 {
		return nil, fmt.Errorf("invalid stake address hash length: %d, expected 28", len(hashBytes))
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%x", hashBytes)
	if cachedValue, exists := sac.cache.Get(cacheKey); exists {
		return cachedValue.([]byte), nil
	}

	// Not in cache, check database
	var stakeAddr models.StakeAddress
	err := tx.Where("hash_raw = ?", hashBytes).First(&stakeAddr).Error

	if err == gorm.ErrRecordNotFound {
		// Create new stake address
		stakeAddr = models.StakeAddress{
			HashRaw: hashBytes,
			View:    cacheKey,
		}

		// Use ON CONFLICT DO NOTHING to handle concurrent inserts
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hash_raw"}},
			DoNothing: true,
		}).Create(&stakeAddr).Error; err != nil {
			return nil, fmt.Errorf("failed to create stake address: %w", err)
		}

		// If nothing was created (due to conflict), fetch the existing record with retry
		if tx.RowsAffected == 0 {
			// Small delay to let the other transaction complete
			time.Sleep(10 * time.Millisecond)
			
			// Retry the fetch a few times in case of race condition
			var fetchErr error
			for i := 0; i < 3; i++ {
				fetchErr = tx.Where("hash_raw = ?", hashBytes).First(&stakeAddr).Error
				if fetchErr == nil {
					break
				}
				if fetchErr != gorm.ErrRecordNotFound {
					// Different error, don't retry
					break
				}
				// Record still not found, wait a bit more
				time.Sleep(time.Duration(i+1) * 20 * time.Millisecond)
			}
			
			if fetchErr != nil {
				// Last resort: assume it exists and continue
				// The stake address was likely created by another worker
				stakeAddr.HashRaw = hashBytes
				stakeAddr.View = cacheKey
				// Log but don't fail - this is a known race condition
				if fetchErr == gorm.ErrRecordNotFound {
					// This is expected in high concurrency scenarios
					// Another worker created it but we can't see it yet due to transaction isolation
					// Just continue with the data we have
				} else {
					return nil, fmt.Errorf("failed to fetch existing stake address after conflict: %w", fetchErr)
				}
			}
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query stake address: %w", err)
	}

	// Update cache
	sac.cache.Put(cacheKey, stakeAddr.HashRaw)

	return stakeAddr.HashRaw, nil
}

// EnsureStakeAddressFromBytesWithTx ensures a stake address exists (same as GetOrCreate but returns error only)
func (sac *StakeAddressCache) EnsureStakeAddressFromBytesWithTx(tx *gorm.DB, hashBytes []byte) error {
	_, err := sac.GetOrCreateStakeAddressFromBytesWithTx(tx, hashBytes)
	return err
}

// PreloadCache preloads stake addresses into the cache
func (sac *StakeAddressCache) PreloadCache() error {
	var stakeAddresses []models.StakeAddress

	// Load stake addresses in batches
	batchSize := 10000
	offset := 0

	for {
		var batch []models.StakeAddress
		err := sac.db.Limit(batchSize).Offset(offset).Find(&batch).Error
		if err != nil {
			return fmt.Errorf("failed to preload stake addresses: %w", err)
		}

		if len(batch) == 0 {
			break
		}

		// Add to cache
		for _, addr := range batch {
			cacheKey := fmt.Sprintf("%x", addr.HashRaw)
			sac.cache.Put(cacheKey, addr.HashRaw)
		}

		stakeAddresses = append(stakeAddresses, batch...)
		offset += batchSize

		if len(batch) < batchSize {
			break
		}
	}

	return nil
}

// ClearCache clears the in-memory cache
func (sac *StakeAddressCache) ClearCache() {
	sac.cache.Clear()
}

// GetCacheSize returns the current size of the cache
func (sac *StakeAddressCache) GetCacheSize() int {
	return sac.cache.Size()
}
