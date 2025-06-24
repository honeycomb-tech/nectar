package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Config represents the complete Nectar configuration
type Config struct {
	Version     string            `toml:"version"`
	Database    DatabaseConfig    `toml:"database"`
	Cardano     CardanoConfig     `toml:"cardano"`
	Performance PerformanceConfig `toml:"performance"`
	Dashboard   DashboardConfig   `toml:"dashboard"`
	Monitoring  MonitoringConfig  `toml:"monitoring"`
	Metadata    MetadataConfig    `toml:"metadata"`
	StateQuery  StateQueryConfig  `toml:"state_query"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	DSN             string        `toml:"dsn"`
	ConnectionPool  int           `toml:"connection_pool"`
	MaxIdleConns    int           `toml:"max_idle_conns"`
	MaxOpenConns    int           `toml:"max_open_conns"`
	ConnMaxLifetime time.Duration `toml:"conn_max_lifetime"`
}

// CardanoConfig holds Cardano node settings
type CardanoConfig struct {
	NodeSocket   string        `toml:"node_socket"`
	NetworkMagic uint32        `toml:"network_magic"`
	ProtocolMode string        `toml:"protocol_mode"`
	Rewards      RewardsConfig `toml:"rewards"`
}

// RewardsConfig holds reward calculation parameters
type RewardsConfig struct {
	TreasuryTax       float64 `toml:"treasury_tax"`
	MonetaryExpansion float64 `toml:"monetary_expansion"`
	OptimalPoolCount  int     `toml:"optimal_pool_count"`
}

// PerformanceConfig holds performance tuning settings
type PerformanceConfig struct {
	WorkerCount        int           `toml:"worker_count"`
	BulkModeEnabled    bool          `toml:"bulk_mode_enabled"`
	BulkFetchRangeSize int           `toml:"bulk_fetch_range_size"`
	StatsInterval      time.Duration `toml:"stats_interval"`
	BlockQueueSize     int           `toml:"block_queue_size"`
}

// DashboardConfig holds dashboard settings
type DashboardConfig struct {
	Enabled     bool   `toml:"enabled"`
	Type        string `toml:"type"`
	WebPort     int    `toml:"web_port"`
	DetailedLog bool   `toml:"detailed_log"`
}

// MonitoringConfig holds monitoring and logging settings
type MonitoringConfig struct {
	MetricsEnabled bool   `toml:"metrics_enabled"`
	MetricsPort    int    `toml:"metrics_port"`
	LogLevel       string `toml:"log_level"`
	LogFormat      string `toml:"log_format"`
}

// MetadataConfig holds metadata fetcher settings
type MetadataConfig struct {
	Enabled     bool   `toml:"enabled"`
	WorkerCount int    `toml:"worker_count"`
	QueueSize   int    `toml:"queue_size"`
	RateLimit   int    `toml:"rate_limit"`
	MaxRetries  int    `toml:"max_retries"`
	UserAgent   string `toml:"user_agent"`
}

// StateQueryConfig holds state query service settings
type StateQueryConfig struct {
	Enabled    bool   `toml:"enabled"`
	SocketPath string `toml:"socket_path"`
}

// Load loads configuration from TOML file with environment variable overrides
func Load(path string) (*Config, error) {
	// Default configuration
	cfg := &Config{
		Version: "1.0",
		Database: DatabaseConfig{
			ConnectionPool:  8,
			MaxIdleConns:    4,
			MaxOpenConns:    8,
			ConnMaxLifetime: time.Hour,
		},
		Cardano: CardanoConfig{
			NodeSocket:   "/opt/cardano/cnode/sockets/node.socket",
			NetworkMagic: 764824073, // Mainnet
			ProtocolMode: "auto",
			Rewards: RewardsConfig{
				TreasuryTax:       0.20,
				MonetaryExpansion: 0.003,
				OptimalPoolCount:  500,
			},
		},
		Performance: PerformanceConfig{
			WorkerCount:        8,
			BulkModeEnabled:    false,
			BulkFetchRangeSize: 2000,
			StatsInterval:      3 * time.Second,
			BlockQueueSize:     10000,
		},
		Dashboard: DashboardConfig{
			Enabled:     true,
			Type:        "terminal",
			WebPort:     8080,
			DetailedLog: false,
		},
		Monitoring: MonitoringConfig{
			MetricsEnabled: true,
			MetricsPort:    9090,
			LogLevel:       "info",
			LogFormat:      "json",
		},
		Metadata: MetadataConfig{
			Enabled:     true,
			WorkerCount: 4,
			QueueSize:   1000,
			RateLimit:   10,
			MaxRetries:  3,
			UserAgent:   "Nectar/1.0",
		},
		StateQuery: StateQueryConfig{
			Enabled:    true,
			SocketPath: "", // Uses cardano.node_socket if empty
		},
	}

	// Load from file if it exists
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			if err := toml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func (c *Config) applyEnvOverrides() {
	// Database overrides
	if dsn := os.Getenv("TIDB_DSN"); dsn != "" {
		c.Database.DSN = dsn
	}
	if dsn := os.Getenv("NECTAR_DSN"); dsn != "" {
		c.Database.DSN = dsn
	}
	if pool := getEnvInt("DB_CONNECTION_POOL"); pool > 0 {
		c.Database.ConnectionPool = pool
	}

	// Cardano overrides
	if socket := os.Getenv("CARDANO_NODE_SOCKET"); socket != "" {
		c.Cardano.NodeSocket = socket
	}
	if magic := getEnvUint32("CARDANO_NETWORK_MAGIC"); magic > 0 {
		c.Cardano.NetworkMagic = magic
	}

	// Performance overrides
	if workers := getEnvInt("WORKER_COUNT"); workers > 0 {
		c.Performance.WorkerCount = workers
	}
	if bulk := getEnvBool("BULK_MODE_ENABLED"); bulk != nil {
		c.Performance.BulkModeEnabled = *bulk
	}
	if size := getEnvInt("BULK_FETCH_RANGE_SIZE"); size > 0 {
		c.Performance.BulkFetchRangeSize = size
	}
	if interval := getEnvDuration("STATS_INTERVAL"); interval > 0 {
		c.Performance.StatsInterval = interval
	}


	// Dashboard overrides
	if dashType := os.Getenv("DASHBOARD_TYPE"); dashType != "" {
		c.Dashboard.Type = dashType
		if dashType == "none" {
			c.Dashboard.Enabled = false
		}
	}
	if port := getEnvInt("WEB_PORT"); port > 0 {
		c.Dashboard.WebPort = port
	}

	// Monitoring overrides
	if enabled := getEnvBool("METRICS_ENABLED"); enabled != nil {
		c.Monitoring.MetricsEnabled = *enabled
	}
	if port := getEnvInt("METRICS_PORT"); port > 0 {
		c.Monitoring.MetricsPort = port
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Monitoring.LogLevel = level
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}
	if c.Database.ConnectionPool <= 0 {
		return fmt.Errorf("database connection pool must be positive")
	}
	if c.Cardano.NodeSocket == "" {
		return fmt.Errorf("cardano node socket is required")
	}
	if c.Performance.WorkerCount <= 0 {
		return fmt.Errorf("worker count must be positive")
	}
	return nil
}

// Helper functions for environment variable parsing
func getEnvInt(key string) int {
	if val := os.Getenv(key); val != "" {
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}

func getEnvUint32(key string) uint32 {
	if val := os.Getenv(key); val != "" {
		i, _ := strconv.ParseUint(val, 10, 32)
		return uint32(i)
	}
	return 0
}

func getEnvBool(key string) *bool {
	if val := os.Getenv(key); val != "" {
		b, _ := strconv.ParseBool(val)
		return &b
	}
	return nil
}

func getEnvDuration(key string) time.Duration {
	if val := os.Getenv(key); val != "" {
		d, _ := time.ParseDuration(val)
		return d
	}
	return 0
}

// Marshal converts a Config struct to TOML bytes
func Marshal(cfg *Config) ([]byte, error) {
	return toml.Marshal(cfg)
}