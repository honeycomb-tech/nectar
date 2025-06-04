# BlockHeader Support Validation Checklist

## Pre-Deployment Validation

### Code Review Checklist
- [ ] `chainSyncRollForwardHandler` handles both `ledger.Block` and `ledger.BlockHeader` types
- [ ] BlockFetch availability checks are in place
- [ ] Connection safety checks implemented
- [ ] Error handling for BlockFetch failures
- [ ] Logging added for both success and failure paths
- [ ] Activity feed integration working
- [ ] Error statistics tracking enabled

### Build Validation
- [ ] Code compiles without errors
- [ ] No linting warnings related to new code
- [ ] Import statements correct
- [ ] Type assertions properly handled

## Deployment Verification

### Initial Startup
- [ ] Nectar starts without errors
- [ ] Connection established successfully
- [ ] Node-to-Node vs Node-to-Client mode detected correctly
- [ ] BlockFetch availability properly detected

### Log Verification
```bash
# Check for BlockHeader-related log messages
tail -f nectar.log | grep -E "(📦|📋|BlockHeader|BlockFetch)"
```

Expected log patterns:
- [ ] `📦 Received full block for slot X (type Y)` - Normal block processing
- [ ] `📋 Received block header for slot X (type Y), fetching full block...` - BlockHeader path
- [ ] `✅ Successfully fetched full block for slot X via BlockFetch` - Successful conversion

### Error Case Validation
- [ ] Node-to-Client mode shows appropriate BlockHeader errors (if any received)
- [ ] Missing BlockFetch shows clear error messages
- [ ] Connection failures handled gracefully

## Runtime Monitoring

### Dashboard Checks
- [ ] Activity feed shows BlockHeader fetch operations
- [ ] Error statistics track BlockFetch failures
- [ ] Sync progress continues normally
- [ ] No "invalid block data type" errors

### Performance Monitoring
- [ ] Blocks per second maintains normal rates
- [ ] No significant performance degradation
- [ ] Memory usage remains stable
- [ ] CPU usage not increased substantially

### Sync Continuity
- [ ] Sync progresses through era boundaries
- [ ] No gaps in block sequence
- [ ] Transaction processing continues normally
- [ ] Database writes successful

## API Validation

### Statistics Endpoint
```bash
curl http://localhost:8080/api/stats | jq '.'
```
- [ ] `blocks_per_second` > 0
- [ ] `blocks_processed` increasing
- [ ] No unusual error counts

### Error Endpoint
```bash
curl http://localhost:8080/api/errors | jq '.'
```
- [ ] No BlockFetch-related errors (unless expected)
- [ ] Error rates within normal ranges

### Activity Feed
```bash
curl http://localhost:8080/api/activity | jq '.'
```
- [ ] BlockHeader activities visible (if any occurred)
- [ ] Timestamps recent and accurate

## Stress Testing

### Mixed Block/BlockHeader Scenario
- [ ] Handle alternating Block and BlockHeader messages
- [ ] No sync interruptions during transitions
- [ ] Performance remains consistent

### BlockFetch Failure Recovery
- [ ] Graceful handling of temporary BlockFetch failures
- [ ] Proper error logging and statistics
- [ ] Sync resumes after BlockFetch recovery

### Era Transition Testing
- [ ] BlockHeader handling works across era boundaries
- [ ] No additional alignment issues introduced
- [ ] Era detection remains accurate

## Troubleshooting Guide

### Common Issues

#### "BlockFetch not available" Errors
- Check Node-to-Node mode: `grep "Node-to-Node connection established" nectar.log`
- Verify `useBlockFetch` flag: Look for BlockFetch configuration logs

#### "BlockFetch client is nil" Errors
- Check connection stability
- Verify node compatibility
- Review connection setup logs

#### High BlockHeader Fetch Rates
- Monitor BlockFetch performance impact
- Check node configuration
- Verify network conditions

### Log Analysis
```bash
# Count Block vs BlockHeader occurrences
grep -c "📦 Received full block" nectar.log
grep -c "📋 Received block header" nectar.log

# Check BlockFetch success rate
grep -c "✅ Successfully fetched" nectar.log
grep -c "❌ Failed to fetch block" nectar.log

# Monitor error patterns
grep "BlockFetch" nectar.log | tail -20
```

## Rollback Plan

### If Issues Occur
1. Stop Nectar immediately
2. Backup current logs for analysis
3. Revert to previous version
4. Restart with previous stable version
5. Document observed issues

### Revert Checklist
- [ ] Previous version binary available
- [ ] Database state preserved
- [ ] Configuration backed up
- [ ] Rollback procedure tested

## Success Criteria

### Primary Objectives
- [ ] No "invalid block data type" errors
- [ ] Sync continues normally with BlockHeaders
- [ ] Performance remains within acceptable ranges
- [ ] Error handling improved

### Secondary Objectives
- [ ] Better alignment with gouroboros behavior
- [ ] Enhanced debugging capabilities
- [ ] Improved error visibility
- [ ] Graceful degradation in edge cases

## Acceptance Sign-off

### Technical Validation
- [ ] All validation steps completed successfully
- [ ] Performance benchmarks met
- [ ] Error handling verified
- [ ] Logging and monitoring functional

### Operational Validation
- [ ] Sync stability confirmed over 24+ hours
- [ ] No unexpected errors or failures
- [ ] Dashboard and APIs responding correctly
- [ ] Database integrity maintained

### Final Approval
- [ ] Development team sign-off
- [ ] Operations team sign-off
- [ ] Documentation updated
- [ ] Monitoring alerts configured

---

**Validation Date:** _____________
**Validator:** _____________
**Environment:** _____________
**Version:** _____________