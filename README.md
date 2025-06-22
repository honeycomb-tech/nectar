# Nectar - High-Performance Cardano Indexer

Nectar is a high-performance Cardano blockchain indexer optimized for TiDB, designed to provide efficient real-time blockchain data processing with comprehensive dashboards.

## Features

- **High-Performance Indexing**: Optimized for TiDB with bulk operations
- **Multi-Mode Dashboard**: Terminal, Web, or both simultaneously
- **Real-Time Updates**: WebSocket support for live data streaming
- **Comprehensive Error System**: Unified error tracking and categorization
- **Era-Aware Processing**: Support for all Cardano eras (Byron through Conway)

## Quick Start

### Prerequisites

- Go 1.21+
- TiDB 7.5.1+ (or MySQL 8.0+)
- Running Cardano node
- 88GB+ RAM and 4+ CPU cores (recommended)

### Installation

1. Clone and build:
```bash
git clone <repository>
cd Nectar
go build -o nectar
```

2. Initialize configuration:
```bash
# Interactive setup (recommended)
./nectar init

# Or non-interactive with defaults
./nectar init --no-interactive
```

3. Edit configuration:
```bash
# Edit nectar.toml with your settings
vim nectar.toml
```

4. Run:
```bash
./start-nectar.sh
# Or from anywhere: /path/to/Nectar/scripts/start-nectar.sh
```

## Configuration

Nectar uses a TOML configuration file (`nectar.toml`). You can also use environment variables or `.env` file for backward compatibility.

### Configuration File (nectar.toml)

```toml
[database]
dsn = "root:password@tcp(localhost:4000)/nectar?charset=utf8mb4&parseTime=True"
connection_pool = 8

[cardano]
node_socket = "/opt/cardano/cnode/sockets/node.socket"
network_magic = 764824073  # Mainnet

[dashboard]
enabled = true
type = "both"  # Options: terminal, web, both, none
web_port = 8080

[performance]
worker_count = 8
bulk_mode_enabled = true
```

### Environment Variables (Alternative)

For backward compatibility, you can still use `.env` file:

```bash
# Database
TIDB_DSN=user:pass@tcp(host:port)/nectar?charset=utf8mb4&parseTime=True&loc=Local

# Cardano Node
CARDANO_NODE_SOCKET=/path/to/node.socket
CARDANO_NETWORK_MAGIC=764824073  # Mainnet

# Dashboard
DASHBOARD_TYPE=both
WEB_PORT=8080

# Performance
BULK_MODE_ENABLED=true
```

## Dashboard Access

- **Terminal**: Progress indicator in console
- **Web**: http://localhost:8080
  - Real-time sync status
  - Era progress visualization
  - Activity feed
  - Error monitoring
  - Performance metrics

## Migrating from .env to TOML

```bash
# Automatically convert .env to nectar.toml
./nectar migrate-env
```

## Testing

Run the indexer and verify the dashboard endpoints are working by accessing the web interface at http://localhost:8080

## Production Deployment

### Systemd Service

```bash
sudo cp nectar.service /etc/systemd/system/
sudo systemctl enable nectar
sudo systemctl start nectar
```

### Docker (Optional)

Note: TiDB runs natively, not in Docker. This is only for Nectar itself if desired:
```bash
docker-compose up -d
```

### Monitoring

- Logs: `journalctl -u nectar -f`
- Error logs: `unified_errors.log`
- Web dashboard: http://your-server:8080

## Architecture

- **main.go**: Entry point and indexer implementation
- **config/**: Configuration system and initialization
- **processors/**: Blockchain data processors (blocks, certificates, assets)
- **models/**: Database models
- **dashboard/**: Dashboard interfaces
- **web/**: Web server and API handlers
- **errors/**: Unified error system
- **statequery/**: Cardano node state queries

## Troubleshooting

1. **Dashboard not loading**: Check `dashboard.type` in nectar.toml (or `DASHBOARD_TYPE` in .env)
2. **Connection refused**: Verify port 8080 is not in use
3. **Database errors**: Check TiDB connection and credentials in nectar.toml
4. **Node connection**: Verify socket path and permissions
5. **Config not found**: Run `./nectar init` to create nectar.toml
6. **Migration from .env**: Run `./nectar migrate-env` to convert .env to nectar.toml

## Development

Build and run with detailed logging:
```bash
go build -tags debug -o nectar
DETAILED_LOG=true ./nectar
```

## License

[Your License Here]