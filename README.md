# Nectar - Cardano Blockchain Indexer 🍯

**Nectar** is a **high-accuracy** Cardano blockchain indexer built in Go, designed for **100% data accuracy** and **reliable sequential processing**. Uses gouroboros for block ingestion and TiDB for distributed storage, with a sequential processing architecture aligned with the reference implementation.

## 📊 Performance

- **Sustained Rate**: 544+ blocks/sec
- **Peak Performance**: 1,000+ blocks/sec  
- **Architecture**: Sequential processing (gouroboros-aligned)
- **Accuracy**: 100% (no race conditions, no dropped blocks)

## 🏗️ Architecture

```
[Public Relays] → [Local Cardano Node] → [Socket] → [Nectar] → [Sequential Processor] → [TiDB]
      ↓                    ↓                           ↓
[IOG/CF Relays]     [Community Relays]         [Block/BlockHeader Handler]
[9 Global Nodes]    [Geographic Distribution]         ↓
                                              [Sequential Transaction Processing]
                                                        ↓
                                              [Direct Database Writes]
```

- **Public Relays**: Connected to 3 official IOG/CF relays + 6 community pool relays globally distributed
- **Local Cardano Node**: Full node providing socket interface for Nectar
- **gouroboros**: Official Cardano protocol implementation
- **Sequential Processing**: One block at a time for maximum accuracy
- **TiDB**: Distributed SQL database for scalability and real-time analytics
- **Complete Schema**: 107 tables covering all Cardano features including governance

## 🌐 Network Connectivity

### **Relay Infrastructure**
Nectar connects through a local Cardano node to a globally distributed network of public relays:

**Official Bootstrap Relays:**
- `backbone.cardano.iog.io:3001` (IOG Official)
- `backbone.mainnet.emurgornd.com:3001` (EMURGO)
- `backbone.mainnet.cardanofoundation.org:3001` (Cardano Foundation)

**Community Pool Relays:** 6 globally distributed community pool relays

**Connection Strategy:**
- 5 active connections + 9 warm backups
- Geographic distribution: North America, Europe, Australia

### **Socket Detection**
Nectar automatically detects and connects to available Cardano node sockets:
- `/opt/cardano/cnode/sockets/node.socket` (primary)
- `/root/workspace/cardano-node-guild/socket/node.socket` (guild tools)
- Environment variable override: `CARDANO_NODE_SOCKET`

## 🚀 Features

### ✅ **Current Capabilities**
- Sequential processing (gouroboros-aligned)
- BlockHeader support for all block types
- Correct era boundaries and smart resuming
- Complete GORM models (107 tables)
- Multi-era support (Byron through Conway)
- TiDB integration with comprehensive error handling

### ✅ **Key Features**
- Sequential processing for 100% accuracy
- BlockHeader support for all sync scenarios
- Correct era boundary handling
- Smart connection with automatic fallback
- Real-time monitoring and error tracking

### 🎯 **Governance Focus** (CIP-1694)
- **Voting Procedures**: Real-time voting data with role-based filtering
- **DRep Management**: Delegation representative tracking and analytics  
- **Committee Operations**: Constitutional committee registration/deregistration
- **Governance Actions**: Proposals, treasury withdrawals, parameter changes
- **Off-chain Data**: Metadata, anchors, and external references



## 📁 Project Structure

```
nectar/
├── main.go             # Sequential processing implementation
├── models/             # GORM models (107 tables)
│   ├── core.go        # Blocks, transactions, basic structures
│   ├── staking.go     # Pools, delegations, rewards
│   ├── governance.go  # CIP-1694 governance features
│   ├── assets.go      # Multi-assets, scripts, datums
│   └── offchain.go    # Metadata and external data
├── database/          # TiDB connection and migrations  
│   └── tidb.go
├── processors/        # Block processing logic
│   ├── block_processor.go
│   └── sequential_block_processor.go
├── migrations/        # SQL migration files (107 files)
├── BLOCKHEADER_SUPPORT.md    # BlockHeader implementation docs
├── BLOCKHEADER_VALIDATION.md # Validation checklist
├── clear_tidb.sh      # Database reset utility
├── socket_detector.go # Smart socket detection
└── go.mod            # Dependencies
```

## 🛠️ Quick Start

### Prerequisites
- **Go 1.21+**
- **Docker & Docker Compose** (for TiDB)
- **Cardano Node Socket** (via Demeter or local node)
- **500GB+ Storage** (for full mainnet sync)

### 1. Clone Repository
```bash
git clone https://github.com/honeycomb-tech/nectar.git
cd nectar
```

### 2. Start TiDB
```bash
# Start TiDB cluster
docker-compose up -d tidb pd tikv

# Verify TiDB is running
docker-compose ps
```

### 3. Build and Run
```bash
# Download dependencies
go mod download

# Build the indexer
go build -o nectar main.go

# Run with auto-detected socket
./nectar
```

### Expected Performance Output
```
╔══════════════════════════════════════════════════════════════════════════════════╗
║                            🍯 NECTAR INDEXER 🍯                            ║
║                         Cardano Blockchain Indexer                         ║
╚══════════════════════════════════════════════════════════════════════════════════╝

█ PERFORMANCE
┌───────────────────────────────────────────────────────────────────────────────────┐
│ Speed:      544 blocks/sec │ RAM: 75.9GB │ CPU:   433% │
│ Blocks:      13335       │ Runtime:    0h 0m │ Era:    Byron │
│ Slot:        13335     │ Peak:     1034 b/s │ Progress:   0.3% │
└───────────────────────────────────────────────────────────────────────────────────┘

█ ACTIVITY FEED
┌───────────────────────────────────────────────────────────────────────────────────┐
│ 📦 Received full block for slot 13335 (type 1)                              │
│ 📦 Received full block for slot 13336 (type 1)                              │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## 🔧 Configuration

### Environment Variables
```bash
# Socket path (auto-detected if not specified)
CARDANO_NODE_SOCKET="/opt/cardano/cnode/sockets/node.socket"

# Database connection (TiDB default)
TIDB_DSN="root@tcp(localhost:4000)/nectar?charset=utf8mb4&parseTime=True&loc=Local"
```

### Network Requirements
```bash
# Ensure local Cardano node is running and connected to public relays
sudo systemctl status cardano-node

# Verify socket accessibility
ls -la /opt/cardano/cnode/sockets/node.socket

# Check relay connections
netstat -tuln | grep 6000
```

### Processing Configuration
```go
// Sequential processing constants
const (
    DB_CONNECTION_POOL = 1    // Single connection for sequential processing
    STATS_INTERVAL = 3 * time.Second
    BLOCKFETCH_TIMEOUT = 30 * time.Second
)
```

## 📊 Database Schema

### Core Tables (12)
- `blocks`, `txes`, `tx_outs`, `tx_ins` - Basic blockchain structure
- `slot_leaders`, `epoches` - Network governance 
- `scripts`, `data`, `redeemers` - Smart contract data

### Staking Tables (16)  
- `stake_addresses`, `pool_hashes` - Identity management
- `pool_updates`, `delegations`, `rewards` - Staking operations
- `withdrawals`, `epoch_stakes` - Reward distribution

### Governance Tables (19) 🏛️
- `voting_procedures`, `gov_action_proposals` - CIP-1694 voting
- `drep_hashes`, `committee_hashes` - Governance actors
- `constitutions`, `treasury_withdrawals` - Protocol governance

### Asset Tables (10)
- `multi_assets`, `ma_tx_outs`, `ma_tx_mints` - Native tokens
- `collateral_tx_outs`, `tx_metadata` - Transaction details

### Off-chain Tables (9)
- `off_chain_vote_data`, `off_chain_pool_data` - External metadata
- `*_fetch_errors` tables - Error tracking and retry logic

## 🔄 Sync Modes

**Bulk Sync**: BlockFetch mode for historical data (544+ blocks/sec)
**Real-time Sync**: ChainSync mode for live data (200-400 blocks/sec)

Automatic transition from bulk to real-time when approaching chain tip.

## 🧪 Testing & Validation

### Build and Test
```bash
# Build
go build -o nectar .

# Quick validation test
timeout 30s ./nectar
```

### Database Validation
```sql
-- Check block continuity
SELECT slot_no, COUNT(*) 
FROM blocks 
GROUP BY slot_no 
HAVING COUNT(*) > 1;

-- Verify era transitions
SELECT era, MIN(slot_no), MAX(slot_no), COUNT(*)
FROM blocks 
GROUP BY era 
ORDER BY MIN(slot_no);
```

### Performance Monitoring
```bash
# Monitor processing activity
tail -f nectar.log | grep -E "(📦|📋|blocks/sec)"

# Check error statistics
curl http://localhost:8080/api/errors
```


## 🔍 Monitoring & Observability

### Built-in Dashboard
Real-time performance metrics, activity feed, error monitoring, and era progress tracking.

### TiDB Integration
Access TiDB dashboard at http://localhost:2379/dashboard for query analysis and cluster monitoring.

## 🚀 Deployment

### Local Development
```bash
# Start TiDB
docker-compose up -d

# Run indexer
go run main.go
```

### Production Docker
```bash
# Build image
docker build -t nectar:latest .

# Deploy with compose
docker-compose -f docker-compose.prod.yml up -d
```

### Database Management
```bash
# Clear database for fresh sync
./clear_tidb.sh

# Monitor sync progress
watch -n 5 "mysql -h127.0.0.1 -P4000 -uroot -e 'SELECT COUNT(*) as blocks FROM nectar.blocks'"
```


## 📚 Resources

- **gouroboros**: https://github.com/blinklabs-io/gouroboros
- **TiDB Documentation**: https://docs.pingcap.com/tidb/stable
- **Cardano Developer Portal**: https://developers.cardano.org/
- **CIP-1694 Governance**: https://cips.cardano.org/cips/cip1694/
- **GORM Documentation**: https://gorm.io/docs/

## 📞 Support

- **GitHub Issues**: Bug reports and feature requests
- **Documentation**: See BlockHeader support and validation docs
- **Monitoring**: Built-in dashboard and error tracking

---

**Built for accuracy and reliability in the Cardano ecosystem** 🍯