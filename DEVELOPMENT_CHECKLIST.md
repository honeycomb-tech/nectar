# Nectar Development Checklist & Next Steps

## Current Status
- **Sync Progress**: ~7.3M blocks (Alonzo era)
- **Performance**: ~39 blocks/second
- **Architecture**: Node-to-Client mode only (BlockFetch code removed)
- **Database**: Local TiDB instance

## 1. Immediate TODOs While Waiting for Sync

### 1.1 TiDB Serverless Integration
**Current Gap**: Nectar only supports standard MySQL/TiDB connections, not TiDB Serverless

#### TiDB Serverless Connection Requirements:
```bash
mysql --comments \
  -u '3KMG9RoJdmgyYep.root' \
  -h gateway01.us-west-2.prod.aws.tidbcloud.com \
  -P 4000 \
  -D 'test' \
  --ssl-mode=VERIFY_IDENTITY \
  --ssl-ca=/etc/ssl/certs/ca-certificates.crt \
  -p'<PASSWORD>'
```

#### What Needs to be Added:
- [ ] SSL/TLS support in DSN configuration
- [ ] Certificate path configuration option
- [ ] Support for TiDB Serverless connection string format
- [ ] Update interactive_init.go to include SSL options
- [ ] Add config fields for:
  - `ssl_mode` (VERIFY_IDENTITY, REQUIRED, etc.)
  - `ssl_ca` (certificate path)
  - `ssl_cert` (client certificate if needed)
  - `ssl_key` (client key if needed)

### 1.2 Configuration System Enhancements

#### Current Limitations:
- Interactive init only supports basic TCP connections
- No SSL/TLS configuration options
- No support for connection through proxy/gateway
- DSN is built with hardcoded parameters

#### Proposed Enhancements:
```toml
[database]
# Current format:
dsn = "user:pass@tcp(host:port)/db?charset=utf8mb4..."

# Proposed additional fields:
connection_type = "tidb-serverless" # or "tidb-local", "mysql"
ssl_enabled = true
ssl_mode = "VERIFY_IDENTITY"
ssl_ca_path = "/etc/ssl/certs/ca-certificates.crt"
ssl_cert_path = "" # optional
ssl_key_path = "" # optional

# Alternative: Support full custom DSN
use_custom_dsn = true
custom_dsn = "user:pass@tcp(gateway.tidbcloud.com:4000)/db?tls=custom&..."
```

### 1.3 TiOperator Integration

#### Research Needed:
- [ ] Can we deploy TiOperator alongside Nectar?
- [ ] Configuration for custom TiDB clusters
- [ ] Resource requirements for TiOperator
- [ ] Integration with Kubernetes (if required)

#### Implementation Steps:
1. Add TiOperator deployment option to setup
2. Create cluster configuration templates
3. Add cluster management commands
4. Document deployment scenarios

## 2. Code Quality & Documentation

### 2.1 Missing Documentation
- [ ] Architecture decision: Why Node-to-Client only?
- [ ] Performance tuning guide
- [ ] TiDB configuration best practices
- [ ] Deployment guide for production
- [ ] SSL/TLS setup guide

### 2.2 Code Cleanup
- [ ] Remove remaining dead code references
- [ ] Update README with current architecture
- [ ] Add inline documentation for N2C design
- [ ] Create migration guide from old versions

## 3. Performance & Optimization

### 3.1 Post-Sync Optimizations
- [ ] Re-enable metadata fetcher after reaching tip
- [ ] Increase workers to match CPU cores (32)
- [ ] Optimize bulk fetch size for real-time processing
- [ ] Implement adaptive performance tuning

### 3.2 Monitoring & Metrics
- [ ] Add TiDB connection pool metrics
- [ ] Track SSL handshake performance
- [ ] Monitor gateway latency (for serverless)
- [ ] Add alerting for connection issues

## 4. Testing Requirements

### 4.1 Connection Testing
- [ ] Test TiDB Serverless connections
- [ ] Test SSL/TLS configurations
- [ ] Test connection failover
- [ ] Test with various network conditions

### 4.2 Integration Testing
- [ ] Test with TiOperator-managed clusters
- [ ] Test migration from local to serverless
- [ ] Test backup/restore procedures
- [ ] Test at chain tip (real-time processing)

## 5. Deployment Scenarios

### 5.1 Local Development
- Current setup (working)
- Local TiDB with Nectar

### 5.2 TiDB Serverless (TODO)
- Nectar → TiDB Cloud Gateway
- SSL/TLS required
- Higher latency considerations

### 5.3 TiOperator Custom Cluster (TODO)
- Kubernetes deployment
- TiOperator managing TiDB
- Nectar as deployment/statefulset

### 5.4 Hybrid Setup (TODO)
- Local Nectar → Remote TiDB
- VPN/Secure tunnel considerations
- Latency optimization

## 6. Security Considerations

### 6.1 Credential Management
- [ ] Remove passwords from config files
- [ ] Support environment variables for secrets
- [ ] Integrate with secret management systems
- [ ] Add config encryption option

### 6.2 Network Security
- [ ] Enforce SSL/TLS for remote connections
- [ ] Add IP whitelisting support
- [ ] Implement connection rate limiting
- [ ] Add audit logging for connections

## 7. Feature Roadmap

### Phase 1: TiDB Serverless Support (Priority)
1. Add SSL/TLS configuration
2. Update DSN builder
3. Test with TiDB Cloud
4. Update documentation

### Phase 2: Enhanced Configuration
1. Environment variable support
2. Secret management integration
3. Multiple database support
4. Connection pooling optimization

### Phase 3: TiOperator Integration
1. Kubernetes deployment templates
2. Operator configuration
3. Cluster management tools
4. Automated scaling

### Phase 4: Production Readiness
1. High availability setup
2. Disaster recovery procedures
3. Performance benchmarks
4. Operational runbooks

## 8. Quick Wins (Can Do Now)

1. **Update DSN Builder** - Add SSL parameters
2. **Environment Variables** - Support for credentials
3. **Documentation** - Explain N2C architecture
4. **Config Validation** - Check SSL cert paths
5. **Connection Test Tool** - Verify TiDB connectivity

## 9. Questions to Research

1. Does TiDB Serverless support bulk operations?
2. What's the latency impact of SSL/TLS?
3. Can we use connection pooling with serverless?
4. What's the best batch size for serverless?
5. How to handle rate limits on serverless?

## 10. Next Immediate Steps

1. **Create SSL/TLS branch** - Start implementing TLS support
2. **Test TiDB Serverless** - Manual connection test
3. **Update Config Structure** - Add new fields
4. **Document Architecture** - Explain N2C decision
5. **Plan Migration Path** - From local to serverless