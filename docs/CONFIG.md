# Nectar Configuration Guide

Nectar now supports TOML-based configuration files, providing a more structured and maintainable way to configure your indexer compared to environment variables.

## Quick Start

### 1. Initialize Configuration

Generate a default configuration file:

```bash
nectar init
```

This creates `nectar.toml` with sensible defaults. You can specify a custom path:

```bash
nectar init --config /etc/nectar/config.toml
```

### 2. Migrate from Environment Variables

If you have existing environment variables, migrate them to a config file:

```bash
nectar migrate-env
```

### 3. Run with Configuration

```bash
nectar --config nectar.toml
```

## Configuration Structure

The configuration file is organized into logical sections:

### Database Configuration
```toml
[database]
dsn = "root:password@tcp(localhost:4000)/nectar?charset=utf8mb4&parseTime=True"
connection_pool = 8
max_idle_conns = 4
max_open_conns = 8
conn_max_lifetime = "1h"
```

### Cardano Node Settings
```toml
[cardano]
node_socket = "/opt/cardano/cnode/sockets/node.socket"
network_magic = 764824073  # Mainnet
protocol_mode = "auto"

[cardano.rewards]
treasury_tax = 0.20
monetary_expansion = 0.003
optimal_pool_count = 500
```

### Performance Tuning
```toml
[performance]
worker_count = 8
bulk_mode_enabled = false
bulk_fetch_range_size = 2000
stats_interval = "3s"
blockfetch_timeout = "30s"
block_queue_size = 10000
```

### Dashboard Settings
```toml
[dashboard]
enabled = true
type = "terminal"  # Options: terminal, web, both, none
web_port = 8080
detailed_log = false
```

### Monitoring
```toml
[monitoring]
metrics_enabled = true
metrics_port = 9090
log_level = "info"  # Options: debug, info, warn, error
log_format = "json"  # Options: json, text
```

## Environment Variable Override

Environment variables still take precedence over config file values for backward compatibility:

- `TIDB_DSN` or `NECTAR_DSN` → overrides `database.dsn`
- `DB_CONNECTION_POOL` → overrides `database.connection_pool`
- `WORKER_COUNT` → overrides `performance.worker_count`
- `DASHBOARD_TYPE` → overrides `dashboard.type`
- etc.

## Multiple Configurations

You can maintain different configurations for different environments:

```bash
# Development
nectar --config dev.toml

# Production
nectar --config production.toml

# Testing
nectar --config test.toml
```

## Command Reference

### `nectar init`
Initialize a new configuration file with defaults.

Options:
- `--config PATH` - Specify where to create the config file (default: `nectar.toml`)

### `nectar migrate-env`
Create a configuration file from current environment variables.

Options:
- `--config PATH` - Specify where to create the config file (default: `nectar.toml`)

### `nectar --config PATH`
Run the indexer with the specified configuration file.

### `nectar version`
Display version information.

### `nectar help`
Show help information.

## Migration Guide

To migrate from environment variables to TOML configuration:

1. Run `nectar migrate-env` to generate a config from your current environment
2. Review and edit the generated file
3. Test with `nectar --config nectar.toml`
4. Once verified, you can remove the environment variables

## Best Practices

1. **Version Control**: Keep your configuration files in version control
2. **Secrets**: Use environment variables for sensitive data like passwords
3. **Comments**: Document your configuration with TOML comments
4. **Validation**: The config loader validates required fields on startup
5. **Defaults**: Unspecified values use sensible defaults

## Example Configurations

### Minimal Configuration
```toml
version = "1.0"

[database]
dsn = "root:password@tcp(localhost:4000)/nectar"

[cardano]
node_socket = "/opt/cardano/cnode/sockets/node.socket"
```

### High-Performance Configuration
```toml
version = "1.0"

[database]
dsn = "root:password@tcp(tidb-cluster:4000)/nectar"
connection_pool = 16
max_open_conns = 16

[performance]
worker_count = 16
bulk_mode_enabled = true
bulk_fetch_range_size = 5000
block_queue_size = 20000

[dashboard]
type = "web"
web_port = 8080
```

### Development Configuration
```toml
version = "1.0"

[database]
dsn = "root:@tcp(localhost:4000)/nectar_dev"
connection_pool = 4

[performance]
worker_count = 4

[dashboard]
type = "terminal"
detailed_log = true

[monitoring]
log_level = "debug"
log_format = "text"
```