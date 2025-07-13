#!/bin/bash
# Comprehensive monitoring script for Nectar + TiDB

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database credentials
DB_USER="root"
DB_PASS="50bcT*DaeU2-19@+Q4"
DB_HOST="127.0.0.1"
DB_PORT="4100"

clear

while true; do
    echo -e "${BLUE}=== NECTAR + TIDB MONITORING DASHBOARD ===${NC}"
    echo "Last Update: $(date)"
    echo ""
    
    # System Resources
    echo -e "${YELLOW}[SYSTEM RESOURCES]${NC}"
    # Disk
    DISK_INFO=$(df -h / | tail -1)
    DISK_USED=$(echo $DISK_INFO | awk '{print $5}' | sed 's/%//')
    DISK_FREE=$(echo $DISK_INFO | awk '{print $4}')
    if [ $DISK_USED -gt 90 ]; then
        echo -e "Disk: ${RED}$DISK_FREE free ($DISK_USED% used) ⚠️${NC}"
    else
        echo -e "Disk: ${GREEN}$DISK_FREE free ($DISK_USED% used)${NC}"
    fi
    
    # Memory
    MEM_INFO=$(free -h | grep Mem)
    MEM_TOTAL=$(echo $MEM_INFO | awk '{print $2}')
    MEM_USED=$(echo $MEM_INFO | awk '{print $3}')
    MEM_FREE=$(echo $MEM_INFO | awk '{print $4}')
    echo "Memory: $MEM_USED/$MEM_TOTAL used, $MEM_FREE free"
    
    # CPU & Load
    LOAD=$(uptime | awk -F'load average:' '{print $2}')
    CPU_COUNT=$(nproc)
    LOAD_1MIN=$(echo $LOAD | awk -F',' '{print $1}' | xargs)
    if (( $(echo "$LOAD_1MIN > $CPU_COUNT" | bc -l) )); then
        echo -e "Load Average:${RED}$LOAD (${CPU_COUNT} CPUs) ⚠️${NC}"
    else
        echo -e "Load Average:${GREEN}$LOAD (${CPU_COUNT} CPUs)${NC}"
    fi
    echo ""
    
    # TiDB Cluster Status
    echo -e "${YELLOW}[TIDB CLUSTER]${NC}"
    # Check if all components are up
    CLUSTER_STATUS=$(tiup cluster display nectar-cluster 2>/dev/null | grep -E "tikv|tidb|pd" | grep -v "Up" | wc -l)
    if [ $CLUSTER_STATUS -eq 0 ]; then
        echo -e "Status: ${GREEN}All components running ✓${NC}"
    else
        echo -e "Status: ${RED}Some components down ✗${NC}"
    fi
    
    # TiKV disk usage
    TIKV_TOTAL=$(du -ch /tidb-data/tikv-* 2>/dev/null | tail -1 | awk '{print $1}')
    echo "TiKV Storage: $TIKV_TOTAL total"
    
    # Show individual TiKV sizes
    du -sh /tidb-data/tikv-* 2>/dev/null | awk '{printf "  %s: %s\n", $2, $1}'
    echo ""
    
    # Nectar Status
    echo -e "${YELLOW}[NECTAR SYNC]${NC}"
    NECTAR_PID=$(ps aux | grep "./nectar" | grep -v grep | awk '{print $2}' | head -1)
    if [ -n "$NECTAR_PID" ]; then
        NECTAR_CPU=$(ps aux | grep -E "^[^ ]*[ ]*$NECTAR_PID" | awk '{print $3}')
        NECTAR_MEM=$(ps aux | grep -E "^[^ ]*[ ]*$NECTAR_PID" | awk '{print $4}')
        echo -e "Status: ${GREEN}Running (PID: $NECTAR_PID)${NC}"
        echo "Resource Usage: CPU: ${NECTAR_CPU}%, Memory: ${NECTAR_MEM}%"
    else
        echo -e "Status: ${RED}Not running ✗${NC}"
    fi
    
    # Database sync progress
    SYNC_INFO=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
        SELECT 
            MAX(slot_no) as last_slot,
            MAX(block_no) as last_block,
            MAX(epoch_no) as last_epoch,
            DATE_FORMAT(MAX(time), '%Y-%m-%d %H:%i') as last_time
        FROM nectar.blocks;" 2>/dev/null)
    
    if [ -n "$SYNC_INFO" ]; then
        LAST_SLOT=$(echo $SYNC_INFO | awk '{print $1}')
        LAST_BLOCK=$(echo $SYNC_INFO | awk '{print $2}')
        LAST_EPOCH=$(echo $SYNC_INFO | awk '{print $3}')
        LAST_TIME=$(echo $SYNC_INFO | awk '{print $4" "$5}')
        
        echo "Last Block: #$LAST_BLOCK (Slot: $LAST_SLOT, Epoch: $LAST_EPOCH)"
        echo "Block Time: $LAST_TIME"
        
        # Calculate sync percentage (mainnet tip ~135M slots)
        MAINNET_TIP=135000000
        SYNC_PCT=$(echo "scale=2; $LAST_SLOT * 100 / $MAINNET_TIP" | bc)
        echo -e "Sync Progress: ${BLUE}${SYNC_PCT}%${NC} of mainnet"
    fi
    
    # Processing rate (last 5 minutes)
    RATE_INFO=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
        SELECT COUNT(*) as blocks
        FROM nectar.blocks 
        WHERE time > DATE_SUB(NOW(), INTERVAL 5 MINUTE);" 2>/dev/null)
    
    if [ -n "$RATE_INFO" ] && [ "$RATE_INFO" -gt 0 ]; then
        BLOCKS_PER_MIN=$(echo "scale=1; $RATE_INFO / 5" | bc)
        echo "Processing Rate: $BLOCKS_PER_MIN blocks/minute"
    fi
    echo ""
    
    # Database Stats
    echo -e "${YELLOW}[DATABASE STATS]${NC}"
    DB_SIZE=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
        SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024 / 1024, 2)
        FROM information_schema.tables 
        WHERE table_schema = 'nectar';" 2>/dev/null)
    echo "Database Size: ${DB_SIZE}GB"
    
    # Table counts
    TABLE_COUNTS=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
        SELECT 
            (SELECT COUNT(*) FROM nectar.blocks) as blocks,
            (SELECT COUNT(*) FROM nectar.txes) as txs,
            (SELECT COUNT(*) FROM nectar.tx_outs) as outputs;" 2>/dev/null)
    
    if [ -n "$TABLE_COUNTS" ]; then
        BLOCKS=$(echo $TABLE_COUNTS | awk '{print $1}')
        TXS=$(echo $TABLE_COUNTS | awk '{print $2}')
        OUTPUTS=$(echo $TABLE_COUNTS | awk '{print $3}')
        echo "Records: $(printf "%'d" $BLOCKS) blocks, $(printf "%'d" $TXS) txs, $(printf "%'d" $OUTPUTS) outputs"
    fi
    echo ""
    
    # Alerts
    echo -e "${YELLOW}[ALERTS]${NC}"
    ALERTS=0
    
    # Check disk space
    if [ $DISK_USED -gt 90 ]; then
        echo -e "${RED}⚠️  Disk space critical: Only $DISK_FREE free${NC}"
        ALERTS=$((ALERTS + 1))
    fi
    
    # Check if Nectar is running
    if [ -z "$NECTAR_PID" ]; then
        echo -e "${RED}⚠️  Nectar is not running${NC}"
        ALERTS=$((ALERTS + 1))
    fi
    
    # Check for recent blocks (no blocks in last 10 minutes = stuck)
    RECENT_BLOCKS=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
        SELECT COUNT(*) FROM nectar.blocks 
        WHERE time > DATE_SUB(NOW(), INTERVAL 10 MINUTE);" 2>/dev/null)
    
    if [ "$RECENT_BLOCKS" = "0" ] && [ -n "$NECTAR_PID" ]; then
        echo -e "${RED}⚠️  No new blocks in last 10 minutes - sync may be stuck${NC}"
        ALERTS=$((ALERTS + 1))
    fi
    
    if [ $ALERTS -eq 0 ]; then
        echo -e "${GREEN}✓ All systems operational${NC}"
    fi
    
    echo ""
    echo "Press Ctrl+C to exit. Refreshing in 30 seconds..."
    sleep 30
    clear
done