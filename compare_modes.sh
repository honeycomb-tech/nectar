#!/bin/bash

# NECTAR MODE COMPARISON SCRIPT
# Compare Concurrent vs Sequential (Gouroboros-aligned) Processing

set -e

echo "🔬 NECTAR MODE COMPARISON - CONCURRENT vs SEQUENTIAL"
echo "===================================================="
echo ""

# Configuration
SOCKET_PATH="/root/workspace/cardano-node-guild/socket/node.socket"
TEST_DURATION=300  # 5 minutes per test
CONCURRENT_LOG="concurrent_test.log"
SEQUENTIAL_LOG="sequential_test.log"
RESULTS_FILE="comparison_results.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}$1${NC}"
    echo "$(printf '=%.0s' {1..60})"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_header "CHECKING PREREQUISITES"
    
    if [ ! -S "$SOCKET_PATH" ]; then
        print_error "Cardano node socket not found at: $SOCKET_PATH"
        exit 1
    fi
    print_success "Cardano node socket found"
    
    if ! command -v go &> /dev/null; then
        print_error "Go not installed"
        exit 1
    fi
    print_success "Go compiler available"
    
    if [ ! -f "go.mod" ]; then
        print_error "Not in Nectar directory (go.mod not found)"
        exit 1
    fi
    print_success "In Nectar project directory"
    
    echo ""
}

# Build both versions
build_indexers() {
    print_header "BUILDING INDEXERS"
    
    echo "🔨 Building concurrent indexer..."
    go build -o nectar_concurrent main.go socket_detector.go enhanced_blockfetch.go
    print_success "Concurrent indexer built"
    
    echo "🔨 Building sequential indexer..."
    go build -o nectar_sequential main_sequential.go socket_detector.go
    print_success "Sequential indexer built"
    
    echo ""
}

# Get current block count from database
get_block_count() {
    mysql -u nectar -pnectar123 -D nectar -e "SELECT COUNT(*) as total_blocks FROM blocks;" 2>/dev/null | tail -n 1
}

# Get latest slot from database
get_latest_slot() {
    mysql -u nectar -pnectar123 -D nectar -e "SELECT MAX(slot_no) as latest_slot FROM blocks WHERE slot_no IS NOT NULL;" 2>/dev/null | tail -n 1
}

# Get transaction count
get_tx_count() {
    mysql -u nectar -pnectar123 -D nectar -e "SELECT COUNT(*) as total_txs FROM txes;" 2>/dev/null | tail -n 1
}

# Reset database for clean test
reset_database() {
    print_header "RESETTING DATABASE FOR CLEAN TEST"
    
    echo "🗑️  Clearing existing data..."
    mysql -u nectar -pnectar123 -D nectar -e "
        SET FOREIGN_KEY_CHECKS = 0;
        TRUNCATE TABLE tx_ins;
        TRUNCATE TABLE tx_outs;
        TRUNCATE TABLE txes;
        TRUNCATE TABLE blocks;
        TRUNCATE TABLE slot_leaders;
        SET FOREIGN_KEY_CHECKS = 1;
    " 2>/dev/null
    
    print_success "Database reset complete"
    echo ""
}

# Run concurrent test
test_concurrent() {
    print_header "TESTING CONCURRENT MODE (Original Nectar)"
    
    # Reset database
    reset_database
    
    # Record start state
    local start_time=$(date +%s)
    local start_blocks=$(get_block_count)
    local start_txs=$(get_tx_count)
    
    print_success "Starting concurrent indexer for $TEST_DURATION seconds..."
    timeout $TEST_DURATION ./nectar_concurrent > $CONCURRENT_LOG 2>&1 &
    local concurrent_pid=$!
    
    # Wait for test completion
    wait $concurrent_pid 2>/dev/null || true
    
    # Record end state
    local end_time=$(date +%s)
    local end_blocks=$(get_block_count)
    local end_txs=$(get_tx_count)
    local latest_slot=$(get_latest_slot)
    local duration=$((end_time - start_time))
    
    # Calculate stats
    local blocks_processed=$((end_blocks - start_blocks))
    local txs_processed=$((end_txs - start_txs))
    local blocks_per_sec=$(echo "scale=2; $blocks_processed / $duration" | bc -l)
    
    # Save results
    echo "CONCURRENT MODE RESULTS:" > $RESULTS_FILE
    echo "======================" >> $RESULTS_FILE
    echo "Duration: $duration seconds" >> $RESULTS_FILE
    echo "Blocks processed: $blocks_processed" >> $RESULTS_FILE
    echo "Transactions processed: $txs_processed" >> $RESULTS_FILE
    echo "Latest slot: $latest_slot" >> $RESULTS_FILE
    echo "Blocks per second: $blocks_per_sec" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    print_success "Concurrent test completed"
    echo "📊 Blocks processed: $blocks_processed"
    echo "📊 Transactions: $txs_processed"
    echo "📊 Performance: $blocks_per_sec blocks/sec"
    echo "📊 Latest slot: $latest_slot"
    echo ""
    
    # Store for comparison
    CONCURRENT_BLOCKS=$blocks_processed
    CONCURRENT_TXS=$txs_processed
    CONCURRENT_SLOT=$latest_slot
    CONCURRENT_RATE=$blocks_per_sec
}

# Run sequential test
test_sequential() {
    print_header "TESTING SEQUENTIAL MODE (Gouroboros-aligned)"
    
    # Reset database
    reset_database
    
    # Record start state
    local start_time=$(date +%s)
    local start_blocks=$(get_block_count)
    local start_txs=$(get_tx_count)
    
    print_success "Starting sequential indexer for $TEST_DURATION seconds..."
    timeout $TEST_DURATION ./nectar_sequential > $SEQUENTIAL_LOG 2>&1 &
    local sequential_pid=$!
    
    # Wait for test completion
    wait $sequential_pid 2>/dev/null || true
    
    # Record end state
    local end_time=$(date +%s)
    local end_blocks=$(get_block_count)
    local end_txs=$(get_tx_count)
    local latest_slot=$(get_latest_slot)
    local duration=$((end_time - start_time))
    
    # Calculate stats
    local blocks_processed=$((end_blocks - start_blocks))
    local txs_processed=$((end_txs - start_txs))
    local blocks_per_sec=$(echo "scale=2; $blocks_processed / $duration" | bc -l)
    
    # Append results
    echo "SEQUENTIAL MODE RESULTS:" >> $RESULTS_FILE
    echo "======================" >> $RESULTS_FILE
    echo "Duration: $duration seconds" >> $RESULTS_FILE
    echo "Blocks processed: $blocks_processed" >> $RESULTS_FILE
    echo "Transactions processed: $txs_processed" >> $RESULTS_FILE
    echo "Latest slot: $latest_slot" >> $RESULTS_FILE
    echo "Blocks per second: $blocks_per_sec" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    print_success "Sequential test completed"
    echo "📊 Blocks processed: $blocks_processed"
    echo "📊 Transactions: $txs_processed"
    echo "📊 Performance: $blocks_per_sec blocks/sec"
    echo "📊 Latest slot: $latest_slot"
    echo ""
    
    # Store for comparison
    SEQUENTIAL_BLOCKS=$blocks_processed
    SEQUENTIAL_TXS=$txs_processed
    SEQUENTIAL_SLOT=$latest_slot
    SEQUENTIAL_RATE=$blocks_per_sec
}

# Compare results
compare_results() {
    print_header "COMPARISON RESULTS"
    
    echo "COMPARISON ANALYSIS:" >> $RESULTS_FILE
    echo "===================" >> $RESULTS_FILE
    
    # Block processing comparison
    if [ "$SEQUENTIAL_BLOCKS" -gt "$CONCURRENT_BLOCKS" ]; then
        local block_diff=$((SEQUENTIAL_BLOCKS - CONCURRENT_BLOCKS))
        local block_improvement=$(echo "scale=1; ($SEQUENTIAL_BLOCKS - $CONCURRENT_BLOCKS) * 100 / $CONCURRENT_BLOCKS" | bc -l)
        print_success "Sequential processed $block_diff more blocks (+${block_improvement}%)"
        echo "Block Processing: Sequential WINS (+$block_improvement%)" >> $RESULTS_FILE
    elif [ "$CONCURRENT_BLOCKS" -gt "$SEQUENTIAL_BLOCKS" ]; then
        local block_diff=$((CONCURRENT_BLOCKS - SEQUENTIAL_BLOCKS))
        local block_improvement=$(echo "scale=1; ($CONCURRENT_BLOCKS - $SEQUENTIAL_BLOCKS) * 100 / $SEQUENTIAL_BLOCKS" | bc -l)
        print_warning "Concurrent processed $block_diff more blocks (+${block_improvement}%)"
        echo "Block Processing: Concurrent WINS (+$block_improvement%)" >> $RESULTS_FILE
    else
        print_success "Equal block processing"
        echo "Block Processing: TIE" >> $RESULTS_FILE
    fi
    
    # Transaction processing comparison
    if [ "$SEQUENTIAL_TXS" -gt "$CONCURRENT_TXS" ]; then
        local tx_diff=$((SEQUENTIAL_TXS - CONCURRENT_TXS))
        local tx_improvement=$(echo "scale=1; ($SEQUENTIAL_TXS - $CONCURRENT_TXS) * 100 / $CONCURRENT_TXS" | bc -l)
        print_success "Sequential processed $tx_diff more transactions (+${tx_improvement}%)"
        echo "Transaction Processing: Sequential WINS (+$tx_improvement%)" >> $RESULTS_FILE
    elif [ "$CONCURRENT_TXS" -gt "$SEQUENTIAL_TXS" ]; then
        local tx_diff=$((CONCURRENT_TXS - SEQUENTIAL_TXS))
        local tx_improvement=$(echo "scale=1; ($CONCURRENT_TXS - $SEQUENTIAL_TXS) * 100 / $SEQUENTIAL_TXS" | bc -l)
        print_warning "Concurrent processed $tx_diff more transactions (+${tx_improvement}%)"
        echo "Transaction Processing: Concurrent WINS (+$tx_improvement%)" >> $RESULTS_FILE
    else
        print_success "Equal transaction processing"
        echo "Transaction Processing: TIE" >> $RESULTS_FILE
    fi
    
    # Performance comparison
    local rate_comparison=$(echo "$SEQUENTIAL_RATE > $CONCURRENT_RATE" | bc -l)
    if [ "$rate_comparison" -eq 1 ]; then
        local rate_improvement=$(echo "scale=1; ($SEQUENTIAL_RATE - $CONCURRENT_RATE) * 100 / $CONCURRENT_RATE" | bc -l)
        print_success "Sequential is faster: $SEQUENTIAL_RATE vs $CONCURRENT_RATE blocks/sec (+${rate_improvement}%)"
        echo "Performance: Sequential WINS (+$rate_improvement%)" >> $RESULTS_FILE
    else
        local rate_improvement=$(echo "scale=1; ($CONCURRENT_RATE - $SEQUENTIAL_RATE) * 100 / $SEQUENTIAL_RATE" | bc -l)
        print_warning "Concurrent is faster: $CONCURRENT_RATE vs $SEQUENTIAL_RATE blocks/sec (+${rate_improvement}%)"
        echo "Performance: Concurrent WINS (+$rate_improvement%)" >> $RESULTS_FILE
    fi
    
    echo ""
    echo "DETAILED COMPARISON:" >> $RESULTS_FILE
    echo "Mode          | Blocks | Transactions | Rate (bl/s) | Latest Slot" >> $RESULTS_FILE
    echo "------------- | ------ | ------------ | ----------- | -----------" >> $RESULTS_FILE
    echo "Concurrent    | $CONCURRENT_BLOCKS | $CONCURRENT_TXS | $CONCURRENT_RATE | $CONCURRENT_SLOT" >> $RESULTS_FILE
    echo "Sequential    | $SEQUENTIAL_BLOCKS | $SEQUENTIAL_TXS | $SEQUENTIAL_RATE | $SEQUENTIAL_SLOT" >> $RESULTS_FILE
    
    # Check for errors in logs
    echo ""
    print_header "ERROR ANALYSIS"
    
    local concurrent_errors=$(grep -i "error\|failed\|panic" $CONCURRENT_LOG 2>/dev/null | wc -l)
    local sequential_errors=$(grep -i "error\|failed\|panic" $SEQUENTIAL_LOG 2>/dev/null | wc -l)
    
    echo "Error Analysis:" >> $RESULTS_FILE
    echo "Concurrent errors: $concurrent_errors" >> $RESULTS_FILE
    echo "Sequential errors: $sequential_errors" >> $RESULTS_FILE
    
    if [ "$sequential_errors" -lt "$concurrent_errors" ]; then
        print_success "Sequential had fewer errors: $sequential_errors vs $concurrent_errors"
        echo "Reliability: Sequential WINS" >> $RESULTS_FILE
    elif [ "$concurrent_errors" -lt "$sequential_errors" ]; then
        print_warning "Concurrent had fewer errors: $concurrent_errors vs $sequential_errors"
        echo "Reliability: Concurrent WINS" >> $RESULTS_FILE
    else
        print_success "Equal error counts: $concurrent_errors"
        echo "Reliability: TIE" >> $RESULTS_FILE
    fi
}

# Generate final report
generate_report() {
    print_header "FINAL REPORT"
    
    echo ""
    echo "📋 Complete results saved to: $RESULTS_FILE"
    echo "📋 Concurrent log: $CONCURRENT_LOG"
    echo "📋 Sequential log: $SEQUENTIAL_LOG"
    echo ""
    
    print_success "Comparison test completed!"
    echo ""
    echo "📊 SUMMARY:"
    echo "  Concurrent: $CONCURRENT_BLOCKS blocks, $CONCURRENT_TXS txs, $CONCURRENT_RATE bl/s"
    echo "  Sequential: $SEQUENTIAL_BLOCKS blocks, $SEQUENTIAL_TXS txs, $SEQUENTIAL_RATE bl/s"
    echo ""
    
    # Recommendation
    local total_sequential=$((SEQUENTIAL_BLOCKS + SEQUENTIAL_TXS))
    local total_concurrent=$((CONCURRENT_BLOCKS + CONCURRENT_TXS))
    
    if [ "$total_sequential" -gt "$total_concurrent" ]; then
        print_success "RECOMMENDATION: Use Sequential Mode (Gouroboros-aligned)"
        echo "RECOMMENDATION: Sequential Mode (better data completeness)" >> $RESULTS_FILE
    else
        print_warning "RECOMMENDATION: Investigate further - Concurrent performed better"
        echo "RECOMMENDATION: Further investigation needed" >> $RESULTS_FILE
    fi
}

# Main execution
main() {
    echo "🔬 Starting comprehensive comparison test..."
    echo "⏱️  Each test will run for $TEST_DURATION seconds"
    echo ""
    
    check_prerequisites
    build_indexers
    test_concurrent
    test_sequential
    compare_results
    generate_report
}

# Run main function
main "$@"