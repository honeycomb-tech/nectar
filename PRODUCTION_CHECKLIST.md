# Nectar Production Deployment Checklist

## Pre-Deployment

### System Requirements
- [ ] CPU: 8+ cores recommended
- [ ] RAM: 16GB+ recommended
- [ ] Storage: 2TB+ SSD for full mainnet history
- [ ] Network: Stable connection to Cardano node
- [ ] OS: Linux (Ubuntu 20.04+ or similar)

### Dependencies
- [ ] Go 1.22+ installed
- [ ] TiDB cluster deployed and accessible
- [ ] Cardano node running and synced
- [ ] Git installed

## Configuration

### Environment Variables
- [ ] `TIDB_DSN` - Database connection string (REQUIRED)
- [ ] `CARDANO_NODE_SOCKET` - Path to Cardano node socket
- [ ] `CARDANO_NETWORK_MAGIC` - Network magic number (764824073 for mainnet)
- [ ] Review performance tuning options in `.env.example`

### Security
- [ ] Database password is strong and unique
- [ ] Database user has minimal required permissions
- [ ] Environment variables stored securely (not in git)
- [ ] TLS enabled for database connections
- [ ] Firewall rules configured appropriately

## Deployment Steps

### Initial Setup
- [ ] Clone repository to production server
- [ ] Copy `.env.example` to `.env`
- [ ] Configure all required environment variables
- [ ] Build binary: `go build -o nectar`
- [ ] Test database connection
- [ ] Verify Cardano node socket accessibility

### Service Installation (Systemd)
- [ ] Copy binary to `/opt/nectar/`
- [ ] Copy migrations to `/opt/nectar/migrations/`
- [ ] Install systemd service file
- [ ] Configure `/etc/nectar/nectar.env`
- [ ] Enable service: `systemctl enable nectar`
- [ ] Start service: `systemctl start nectar`

### Docker Deployment (Alternative)
- [ ] Review `docker-compose.yml`
- [ ] Create `.env` file with configuration
- [ ] Build image: `docker-compose build`
- [ ] Start services: `docker-compose up -d`

## Monitoring Setup

### Logs
- [ ] Configure log rotation
- [ ] Set up log aggregation (if applicable)
- [ ] Monitor `errors.log` and `unified_errors.log`

### Metrics
- [ ] Enable Prometheus metrics (if needed)
- [ ] Configure Grafana dashboards
- [ ] Set up alerting rules

### Health Checks
- [ ] Monitor blocks per second
- [ ] Check database connection pool usage
- [ ] Monitor tip distance (blocks behind)
- [ ] Set up alerts for indexer stoppage

## Performance Validation

### Initial Sync
- [ ] Verify 800-1000 blocks/second performance
- [ ] Monitor memory usage (should be 2-4GB)
- [ ] Check database growth rate
- [ ] Ensure no excessive errors in logs

### Resource Utilization
- [ ] CPU usage reasonable (<80% sustained)
- [ ] Memory stable (no leaks)
- [ ] Database connections within limits
- [ ] Network bandwidth sufficient

## Backup and Recovery

### Database
- [ ] Configure TiDB backup schedule
- [ ] Test restore procedure
- [ ] Document recovery steps

### Configuration
- [ ] Backup environment configuration
- [ ] Document all custom settings
- [ ] Version control for configuration changes

## Post-Deployment

### Verification
- [ ] Indexer syncing successfully
- [ ] No errors in logs
- [ ] Performance metrics acceptable
- [ ] All features working (rewards, governance, etc.)

### Documentation
- [ ] Update internal documentation
- [ ] Record deployment specifics
- [ ] Create runbook for common operations
- [ ] Train operations team

## Maintenance Tasks

### Regular Tasks
- [ ] Monitor disk space usage
- [ ] Review error logs weekly
- [ ] Update gouroboros library quarterly
- [ ] Performance tuning as needed

### Emergency Procedures
- [ ] Document rollback procedure
- [ ] Create incident response plan
- [ ] Test disaster recovery
- [ ] Maintain contact list

## Known Issues

### Current Limitations
- [ ] Byron address panic handling required (defensive code in place)
- [ ] Redeemer data not accessible via gouroboros
- [ ] Reference scripts field exists but not exposed
- [ ] Protocol versions approximated by era

### Workarounds
- [ ] Byron addresses use hex fallback representation
- [ ] Missing features logged but don't stop processing
- [ ] Graceful degradation for unsupported features

## Sign-off

- [ ] Development team approval
- [ ] Operations team trained
- [ ] Security review completed
- [ ] Performance benchmarks met
- [ ] Documentation complete
- [ ] Monitoring configured
- [ ] Backups verified

Date: _________________
Approved by: _________________