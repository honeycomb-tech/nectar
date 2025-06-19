# Nectar Project Status - June 19, 2025

## Current State

### What Was Accomplished
1. **Dashboard System Overhaul**
   - Completely redesigned dashboard architecture with interface-based approach
   - Implemented dual-mode dashboard (terminal + web simultaneously)
   - Fixed all web dashboard rendering issues
   - Fixed error forwarding from unified error system to dashboards
   - Both dashboards now fully functional and display real-time data

2. **Key Fixes Applied**
   - Fixed template loading issue by using manual template parsing instead of LoadHTMLFiles
   - Fixed WebSocket JSON concatenation causing parse errors
   - Fixed template data capitalization mismatches (handlers now use capitalized keys)
   - Fixed startup script to handle special characters in database credentials
   - Fixed rollback handler panic when dealing with genesis block (empty hash)

3. **Database Status**
   - Database has been completely recreated (empty)
   - Ready to sync from genesis
   - All tables created with proper indexes

### Current Configuration
- **Database**: TiDB at 127.0.0.1:4000
- **Cardano Node**: Socket at /root/workspace/cardano-node-guild/socket/node.socket
- **Network**: Mainnet (magic: 764824073)
- **Dashboard**: Both terminal and web enabled
- **Web Port**: 8080

## Next Steps for Error Analysis

### 1. Start Fresh Sync
```bash
./start-nectar.sh
```
This will begin syncing from genesis. Byron era will sync without issues.

### 2. Monitor Shelley Transition
When Nectar reaches slot 4,492,800 (start of Shelley), errors will begin appearing. The key areas to watch:

- **Stake Registration Errors**: Missing stake addresses
- **Pool Registration Errors**: Complex certificate processing
- **Delegation Errors**: References to non-existent stake addresses
- **Withdrawal Processing**: New transaction type handling

### 3. Error Analysis Tools
- **Web Dashboard**: http://localhost:8080 - Visual error monitoring
- **Error Log**: `unified_errors.log` - Detailed error information
- **Database Queries**: Check specific tables for data integrity

## Key Code Locations

### Error Flow
1. **Error Source**: `processors/*_processor.go` - Where errors originate
2. **Error Collection**: `errors/unified_error_system.go` - Central error management
3. **Dashboard Callback**: `main.go:531` - Error forwarding to dashboard
4. **Dashboard Display**: 
   - Terminal: `dashboard/terminal_adapter.go`
   - Web: `web/handlers.go` and templates

### Important Functions
- `chainSyncRollForwardHandler` (main.go:1126) - Processes incoming blocks
- `LogError` (errors/unified_error_system.go) - Central error logging
- `ProcessStakeRegistration` (processors/certificate_processor.go) - Shelley-specific processing

## Common Operations

### Check Database Status
```sql
mysql -h 127.0.0.1 -P 4000 -u root -p'Ds*uS+pG278T@-3K60' nectar -e "
  SELECT COUNT(*) as blocks FROM blocks;
  SELECT MAX(slot_no) as last_slot FROM blocks;
  SELECT COUNT(*) as errors FROM unified_errors;
"
```

### Monitor Sync Progress
- Terminal: Watch the progress indicator
- Web: http://localhost:8080
- Logs: `tail -f unified_errors.log`

### Test Dashboard Endpoints
```bash
./test_dashboard.sh  # If you recreate this script
```

## Known Issues to Address

### Shelley Era Errors (Expected)
1. **Stake Address Creation**: Need to handle stake registrations before delegations
2. **Pool Metadata**: Async fetching causes timing issues
3. **Certificate Order**: Certificates in same transaction must be processed in order
4. **UTXO Tracking**: New transaction types need proper handling

### Performance Considerations
- Bulk operations help but can cause memory pressure
- Error deduplication is working but may need tuning
- Dashboard updates every second - adjust if needed

## Development Workflow

### Making Changes
1. Edit code
2. `go build -o nectar`
3. `./start-nectar.sh`

### Testing Error Fixes
1. Let it sync to Shelley (slot 4,492,800+)
2. Watch error patterns in dashboard
3. Fix specific error types
4. Clear database and resync to verify

### Debugging
- Enable detailed logging: `DETAILED_LOG=true ./nectar`
- Check specific processor: Look for `Log*` variables in processors
- Database queries: Use mysql client to inspect data

## Environment Setup

All configuration is in `.env`:
```bash
TIDB_DSN=root:Ds*uS+pG278T@-3K60@tcp(127.0.0.1:4000)/nectar?...
CARDANO_NODE_SOCKET=/root/workspace/cardano-node-guild/socket/node.socket
CARDANO_NETWORK_MAGIC=764824073
DASHBOARD_TYPE=both
WEB_PORT=8080
BULK_MODE_ENABLED=true
```

## Summary

The project is now in a clean state with working dashboards and error monitoring. The database is empty and ready for a fresh sync. When Shelley era begins (around slot 4.5M), you'll see the errors that need fixing. The unified error system will capture them, deduplicate them, and display them in both dashboards for analysis.

The main focus should be on fixing the certificate processing order and ensuring stake addresses exist before they're referenced in delegations.