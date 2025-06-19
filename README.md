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

2. Configure environment:
```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. Run:
```bash
./start-nectar.sh
```

## Configuration

### Environment Variables (.env)

```bash
# Database
TIDB_DSN=user:pass@tcp(host:port)/nectar?charset=utf8mb4&parseTime=True&loc=Local

# Cardano Node
CARDANO_NODE_SOCKET=/path/to/node.socket
CARDANO_NETWORK_MAGIC=764824073  # Mainnet

# Dashboard
DASHBOARD_TYPE=both  # Options: terminal, web, both, none
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

## Testing

Verify all endpoints are working:
```bash
./test_dashboard.sh
```

## Production Deployment

### Systemd Service

```bash
sudo cp nectar.service /etc/systemd/system/
sudo systemctl enable nectar
sudo systemctl start nectar
```

### Docker

```bash
docker-compose up -d
```

### Monitoring

- Logs: `journalctl -u nectar -f`
- Error logs: `unified_errors.log`
- Web dashboard: http://your-server:8080

## Architecture

- **main.go**: Entry point and indexer implementation
- **dashboard/**: Dashboard interfaces and implementations
- **web/**: Web server, handlers, and templates
- **errors/**: Unified error system
- **models/**: Database models
- **bulk/**: Bulk operation handlers
- **metadata/**: Token metadata fetching

## Troubleshooting

1. **Dashboard not loading**: Check `DASHBOARD_TYPE` in .env
2. **Connection refused**: Verify port 8080 is not in use
3. **Database errors**: Check TiDB connection and credentials
4. **Node connection**: Verify socket path and permissions

## Development

Build and run with detailed logging:
```bash
go build -tags debug -o nectar
DETAILED_LOG=true ./nectar
```

## License

[Your License Here]