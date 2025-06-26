#!/bin/bash

echo "=== NECTAR COMPREHENSIVE DIAGNOSTIC ==="
echo

# 1. Check process status
echo "1. PROCESS STATUS:"
ps aux | grep nectar | grep -v grep | head -2
echo

# 2. Check current sync position
echo "2. CURRENT SYNC POSITION:"
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' nectar -e "
SELECT 
    MAX(slot_no) as current_slot,
    MAX(epoch_no) as current_epoch,
    COUNT(*) as total_blocks,
    NOW() as check_time
FROM blocks;" 2>/dev/null
echo

# 3. Check for slow queries
echo "3. SLOW QUERIES (last 10):"
tail -1000 unified_errors.log 2>/dev/null | grep "SLOW QUERY" | tail -10
echo

# 4. Check StateQuery tables
echo "4. STATEQUERY DATA STATUS:"
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' nectar -e "
SELECT 
    'rewards' as table_name, COUNT(*) as count FROM rewards
UNION ALL SELECT 'pool_stats', COUNT(*) FROM pool_stats
UNION ALL SELECT 'epoch_stakes', COUNT(*) FROM epoch_stakes
UNION ALL SELECT 'treasury', COUNT(*) FROM treasury
UNION ALL SELECT 'reserves', COUNT(*) FROM reserves;" 2>/dev/null
echo

# 5. Check epoch boundaries crossed
echo "5. EPOCH BOUNDARIES CROSSED:"
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' nectar -e "
SELECT 
    epoch_no,
    COUNT(*) as blocks_in_epoch,
    MIN(slot_no) as first_slot,
    MAX(slot_no) as last_slot
FROM blocks 
WHERE epoch_no >= 210
GROUP BY epoch_no
ORDER BY epoch_no DESC
LIMIT 10;" 2>/dev/null
echo

# 6. Check for missing data
echo "6. MISSING DATA CHECK:"
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' nectar -e "
SELECT 
    'Scripts (should have some)' as check_item,
    CASE WHEN COUNT(*) = 0 THEN '❌ MISSING' ELSE CONCAT('✅ ', COUNT(*)) END as status
FROM scripts
UNION ALL
SELECT 
    'Metadata (config disabled)',
    CASE WHEN COUNT(*) = 0 THEN '⚠️ Expected (disabled)' ELSE CONCAT('✅ ', COUNT(*)) END
FROM tx_metadata
UNION ALL
SELECT 
    'Multi-assets (Mary era)',
    CASE WHEN COUNT(*) = 0 THEN '⚠️ Not in Mary yet' ELSE CONCAT('✅ ', COUNT(*)) END
FROM multi_assets;" 2>/dev/null
echo

# 7. Check connection health
echo "7. DATABASE CONNECTIONS:"
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' -e "SHOW PROCESSLIST;" 2>/dev/null | grep nectar | wc -l
echo " active connections"
echo

# 8. Check for epoch processing logs
echo "8. EPOCH PROCESSING LOGS:"
grep -i "epoch.*boundary\|calculating.*reward\|state.*query.*start" nectar.log 2>/dev/null | tail -5 || echo "No epoch logs found"
echo

# 9. Check slot_leaders issue
echo "9. SLOT LEADERS TABLE:"
mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' nectar -e "
SELECT 
    COUNT(*) as total_rows,
    COUNT(DISTINCT hash) as unique_hashes,
    COUNT(DISTINCT pool_hash) as unique_pools,
    SUM(CASE WHEN pool_hash = '' OR pool_hash IS NULL THEN 1 ELSE 0 END) as empty_pool_hash
FROM slot_leaders;" 2>/dev/null
echo

# 10. Check for errors in last hour
echo "10. RECENT ERRORS (last hour):"
find . -name "*.log" -mmin -60 -exec grep -l "ERROR\|FATAL\|panic" {} \; 2>/dev/null | head -5

echo
echo "=== DIAGNOSTIC COMPLETE ==="