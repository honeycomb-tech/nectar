package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ConnectionRecovery handles automatic database connection recovery
type ConnectionRecovery struct {
	dsn               string
	db                *gorm.DB
	mutex             sync.RWMutex
	isHealthy         bool
	lastHealthCheck   time.Time
	reconnectAttempts int
	maxReconnectDelay time.Duration
	healthCheckTicker *time.Ticker
	stopChan          chan struct{}
	onReconnect       func(*gorm.DB) error // Callback for re-initialization after reconnect
}

// NewConnectionRecovery creates a new connection recovery handler
func NewConnectionRecovery(dsn string, onReconnect func(*gorm.DB) error) *ConnectionRecovery {
	return &ConnectionRecovery{
		dsn:               dsn,
		maxReconnectDelay: 30 * time.Second,
		stopChan:          make(chan struct{}),
		onReconnect:       onReconnect,
	}
}

// Start begins the health check monitoring
func (cr *ConnectionRecovery) Start(ctx context.Context) error {
	// Initial connection
	if err := cr.connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start health check routine
	cr.healthCheckTicker = time.NewTicker(5 * time.Second)
	go cr.healthCheckLoop(ctx)

	return nil
}

// Stop stops the health check monitoring
func (cr *ConnectionRecovery) Stop() {
	close(cr.stopChan)
	if cr.healthCheckTicker != nil {
		cr.healthCheckTicker.Stop()
	}
}

// GetDB returns the current database connection with automatic recovery
func (cr *ConnectionRecovery) GetDB() (*gorm.DB, error) {
	cr.mutex.RLock()
	db := cr.db
	healthy := cr.isHealthy
	cr.mutex.RUnlock()

	// If healthy, return immediately
	if healthy && db != nil {
		return db, nil
	}

	// Not healthy, try to recover
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// Double-check after acquiring write lock
	if cr.isHealthy && cr.db != nil {
		return cr.db, nil
	}

	// Attempt reconnection
	if err := cr.reconnectWithBackoff(); err != nil {
		return nil, fmt.Errorf("connection recovery failed: %w", err)
	}

	return cr.db, nil
}

// connect establishes a new database connection
func (cr *ConnectionRecovery) connect() error {
	db, err := gorm.Open(mysql.Open(cr.dsn), &gorm.Config{
		Logger:                                   NewGormLogger(),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Optimized settings for recovery
	sqlDB.SetMaxIdleConns(8)
	sqlDB.SetMaxOpenConns(16)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return fmt.Errorf("ping failed: %w", err)
	}

	// Run callback if provided (for re-initialization)
	if cr.onReconnect != nil {
		if err := cr.onReconnect(db); err != nil {
			sqlDB.Close()
			return fmt.Errorf("onReconnect callback failed: %w", err)
		}
	}

	cr.db = db
	cr.isHealthy = true
	cr.lastHealthCheck = time.Now()
	cr.reconnectAttempts = 0

	log.Println("[ConnectionRecovery] Database connection established successfully")
	return nil
}

// reconnectWithBackoff attempts to reconnect with exponential backoff
func (cr *ConnectionRecovery) reconnectWithBackoff() error {
	baseDelay := time.Second
	maxAttempts := 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > cr.maxReconnectDelay {
				delay = cr.maxReconnectDelay
			}
			log.Printf("[ConnectionRecovery] Waiting %v before reconnect attempt %d/%d", delay, attempt+1, maxAttempts)
			time.Sleep(delay)
		}

		if err := cr.connect(); err != nil {
			log.Printf("[ConnectionRecovery] Reconnect attempt %d/%d failed: %v", attempt+1, maxAttempts, err)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}

// healthCheckLoop continuously monitors connection health
func (cr *ConnectionRecovery) healthCheckLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-cr.stopChan:
			return
		case <-cr.healthCheckTicker.C:
			cr.checkHealth()
		}
	}
}

// checkHealth verifies the connection is still healthy
func (cr *ConnectionRecovery) checkHealth() {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	if cr.db == nil {
		cr.isHealthy = false
		return
	}

	sqlDB, err := cr.db.DB()
	if err != nil {
		log.Printf("[ConnectionRecovery] Failed to get sql.DB: %v", err)
		cr.isHealthy = false
		return
	}

	// Simple ping test
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Printf("[ConnectionRecovery] Health check failed: %v", err)
		cr.isHealthy = false

		// Try to recover immediately
		go func() {
			cr.mutex.Lock()
			defer cr.mutex.Unlock()

			if !cr.isHealthy {
				log.Println("[ConnectionRecovery] Attempting automatic recovery...")
				if err := cr.reconnectWithBackoff(); err != nil {
					log.Printf("[ConnectionRecovery] Automatic recovery failed: %v", err)
				}
			}
		}()
		return
	}

	// Update health status
	cr.isHealthy = true
	cr.lastHealthCheck = time.Now()
}

// IsHealthy returns the current health status
func (cr *ConnectionRecovery) IsHealthy() bool {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	return cr.isHealthy
}

// GetLastHealthCheck returns the time of the last successful health check
func (cr *ConnectionRecovery) GetLastHealthCheck() time.Time {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()
	return cr.lastHealthCheck
}
