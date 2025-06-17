# Nectar Production Readiness Guide

## Overview

Nectar is a high-performance Cardano blockchain indexer optimized for TiDB. This guide covers deployment, configuration, and operational considerations for production environments.

## Configuration

All sensitive configuration should be provided via environment variables:

### Required Environment Variables

```bash
# Database Connection (REQUIRED)
export TIDB_DSN="user:password@tcp(host:port)/nectar?charset=utf8mb4&parseTime=True&loc=Local"

# Cardano Network Configuration (OPTIONAL - defaults shown)
export CARDANO_NETWORK_MAGIC="764824073"  # Mainnet: 764824073, Preprod: 1, Preview: 2
export CARDANO_NODE_SOCKET="/path/to/node.socket"
```

### Optional Environment Variables

```bash
# Performance Tuning
export DB_CONNECTION_POOL="8"          # Database connection pool size (default: 8)
export WORKER_COUNT="8"                # Number of parallel workers (default: 8)
export BULK_MODE_ENABLED="false"       # Enable bulk sync mode (default: false)
export BULK_FETCH_RANGE_SIZE="2000"    # Blocks per bulk fetch (default: 2000)
export STATS_INTERVAL="3s"             # Dashboard update interval (default: 3s)
export BLOCKFETCH_TIMEOUT="30s"        # Block fetch timeout (default: 30s)

# Monitoring and Debugging
export METRICS_ENABLED="true"          # Enable Prometheus metrics
export METRICS_PORT="9090"             # Metrics endpoint port
export DEBUG_MODE="false"              # Enable debug logging
export LOG_LEVEL="info"                # Log level: debug, info, warn, error

# Reward Calculation Parameters (Mainnet defaults)
export CARDANO_TREASURY_TAX="0.20"           # Treasury tax (20%)
export CARDANO_MONETARY_EXPANSION="0.003"    # Monetary expansion per epoch (0.3%)
export CARDANO_OPTIMAL_POOL_COUNT="500"      # Optimal number of pools (k=500)
```

## Database Setup

1. Ensure TiDB cluster is properly configured
2. Create the database: `CREATE DATABASE IF NOT EXISTS nectar`
3. Migrations run automatically on startup

## Known Limitations

### From Gouroboros Library

1. **Redeemer Data**: Not directly accessible through transaction interface
2. **Reference Scripts**: ScriptRef field exists but not exposed via interface
3. **Byron Address Panics**: Some early Byron addresses cause panics (handled gracefully)

### Protocol Versions

Protocol versions are approximated based on era. Actual versions depend on voted parameter updates.

### Performance Considerations

1. **Initial Sync**: ~800-1000 blocks/second with default settings
2. **Memory Usage**: ~2-4GB during normal operation
3. **Database Growth**: ~1TB for full mainnet history

## Monitoring

### Health Checks

- Dashboard available on terminal output
- Error logs: `errors.log` and `unified_errors.log`
- Slow query monitoring built-in

### Key Metrics to Monitor

1. Blocks per second
2. Database connection pool usage
3. Error rates (especially panic recoveries)
4. Queue depths

## Security Considerations

1. **Never commit credentials** to version control
2. Use read-only database users where possible
3. Restrict network access to Cardano node socket
4. Enable TLS for database connections in production

## Troubleshooting

### Common Issues

1. **"Data too long for column"**: Database migration needed
2. **Connection refused**: Check socket path and permissions
3. **Slow sync**: Increase worker count or connection pool

### Debug Mode

Enable detailed logging:
```bash
export DEBUG_MODE="true"
export LOG_LEVEL="debug"
```

## Deployment Checklist

- [ ] Set all required environment variables
- [ ] Verify database connectivity
- [ ] Test Cardano node connection
- [ ] Review resource limits (CPU, memory)
- [ ] Set up monitoring and alerting
- [ ] Configure log rotation
- [ ] Plan for database backups
- [ ] Document your specific configuration

## Support

For issues, improvements, or questions:
- GitHub Issues: [your-repo-url]
- Documentation: See README.md