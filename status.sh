#!/bin/bash
# Quick status check for all components

echo "=== SYSTEM STATUS SUMMARY ==="
echo "Time: $(date)"
echo ""

# Quick disk check
DISK_FREE=$(df -h / | tail -1 | awk '{print $4}')
DISK_USED=$(df -h / | tail -1 | awk '{print $5}')
echo "📁 Disk: $DISK_FREE free ($DISK_USED used)"

# Memory
MEM_FREE=$(free -h | grep Mem | awk '{print $7}')
echo "💾 Memory Available: $MEM_FREE"

# TiKV sizes
TIKV_TOTAL=$(du -ch /tidb-data/tikv-* 2>/dev/null | tail -1 | awk '{print $1}')
echo "🗄️  TiKV Storage: $TIKV_TOTAL"

# Nectar status
if ps aux | grep -q "[.]/nectar"; then
    echo "✅ Nectar: Running"
    # Get last block
    LAST_BLOCK=$(mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' -s -N -e "SELECT MAX(block_no) FROM nectar.blocks;" 2>/dev/null)
    LAST_EPOCH=$(mysql -h 127.0.0.1 -P 4100 -u root -p'50bcT*DaeU2-19@+Q4' -s -N -e "SELECT MAX(epoch_no) FROM nectar.blocks;" 2>/dev/null)
    echo "📊 Last Block: #$LAST_BLOCK (Epoch $LAST_EPOCH)"
else
    echo "❌ Nectar: Not running"
fi

# TiDB cluster
UP_COUNT=$(tiup cluster display nectar-cluster 2>/dev/null | grep -c "Up")
echo "🔧 TiDB Cluster: $UP_COUNT components running"

# Compaction status
BEFORE_SIZE="1,135GB"
CURRENT_SIZE=$TIKV_TOTAL
echo "🗜️  Compaction: $BEFORE_SIZE → $CURRENT_SIZE"

echo ""
echo "For detailed monitoring, run: ./monitor.sh"