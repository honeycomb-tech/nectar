#!/bin/bash

# NECTAR TiDB DATABASE CLEANER
# Clears all data for fresh sequential indexing start

set -e

echo "🗑️  NECTAR DATABASE CLEANER"
echo "=========================="
echo ""

# Database configuration
DB_HOST="127.0.0.1"
DB_PORT="4000"
DB_NAME="nectar"
DB_USER="root"
DB_PASS="Ds*uS+pG278T@-3K60"

# Check if TiDB is accessible
echo "🔍 Checking TiDB connection..."
if ! mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -e "SELECT 1;" 2>/dev/null; then
    echo "❌ Cannot connect to TiDB"
    echo "   Host: $DB_HOST:$DB_PORT"
    echo "   User: $DB_USER"
    echo "   Database: $DB_NAME"
    exit 1
fi

echo "✅ TiDB connection successful"
echo ""

# Get table count before clearing
echo "📊 Current database state:"
BLOCK_COUNT=$(mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME -e "SELECT COUNT(*) as count FROM blocks;" 2>/dev/null | tail -n 1)
TX_COUNT=$(mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME -e "SELECT COUNT(*) as count FROM txes;" 2>/dev/null | tail -n 1)
TXOUT_COUNT=$(mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME -e "SELECT COUNT(*) as count FROM tx_outs;" 2>/dev/null | tail -n 1)
TXIN_COUNT=$(mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME -e "SELECT COUNT(*) as count FROM tx_ins;" 2>/dev/null | tail -n 1)

echo "   Blocks: $BLOCK_COUNT"
echo "   Transactions: $TX_COUNT" 
echo "   TX Outputs: $TXOUT_COUNT"
echo "   TX Inputs: $TXIN_COUNT"
echo ""

# Confirmation prompt
read -p "⚠️  This will DELETE ALL DATA. Continue? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Operation cancelled"
    exit 1
fi

echo ""
echo "🗑️  Clearing database data..."

# Clear data in correct order to avoid foreign key constraints
mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME << 'EOF'
SET FOREIGN_KEY_CHECKS = 0;

-- Clear transaction-related tables first
TRUNCATE TABLE tx_ins;
TRUNCATE TABLE tx_outs;
TRUNCATE TABLE ma_tx_outs;
TRUNCATE TABLE redeemers;
TRUNCATE TABLE tx_metadata;
TRUNCATE TABLE scripts;
TRUNCATE TABLE data;

-- Clear transaction table
TRUNCATE TABLE txes;

-- Clear block-related tables
TRUNCATE TABLE blocks;
TRUNCATE TABLE slot_leaders;

-- Clear staking tables
TRUNCATE TABLE stake_addresses;
TRUNCATE TABLE pool_hashes;
TRUNCATE TABLE epoch_stakes;
TRUNCATE TABLE rewards;
TRUNCATE TABLE withdrawals;
TRUNCATE TABLE stake_registrations;
TRUNCATE TABLE stake_deregistrations;
TRUNCATE TABLE delegations;

-- Clear governance tables  
TRUNCATE TABLE gov_action_proposals;
TRUNCATE TABLE voting_procedures;
TRUNCATE TABLE committee_registrations;

-- Clear asset tables
TRUNCATE TABLE multi_assets;
TRUNCATE TABLE ma_tx_mints;

-- Clear epoch summary
TRUNCATE TABLE epoches;

-- Clear any remaining tables
TRUNCATE TABLE collateral_tx_ins;
TRUNCATE TABLE reference_tx_ins;
TRUNCATE TABLE collateral_tx_outs;

SET FOREIGN_KEY_CHECKS = 1;
EOF

if [ $? -eq 0 ]; then
    echo "✅ Database cleared successfully!"
    echo ""
    
    # Verify tables are empty
    echo "📊 Database state after clearing:"
    NEW_BLOCK_COUNT=$(mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME -e "SELECT COUNT(*) as count FROM blocks;" 2>/dev/null | tail -n 1)
    NEW_TX_COUNT=$(mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p"$DB_PASS" -D$DB_NAME -e "SELECT COUNT(*) as count FROM txes;" 2>/dev/null | tail -n 1)
    
    echo "   Blocks: $NEW_BLOCK_COUNT"
    echo "   Transactions: $NEW_TX_COUNT"
    echo ""
    
    if [ "$NEW_BLOCK_COUNT" = "0" ] && [ "$NEW_TX_COUNT" = "0" ]; then
        echo "🎉 Database successfully cleared - ready for fresh sequential indexing!"
        echo ""
        echo "📋 Summary:"
        echo "   Removed $BLOCK_COUNT blocks"
        echo "   Removed $TX_COUNT transactions"
        echo "   Database ready for clean start"
    else
        echo "⚠️  Warning: Some data may remain"
    fi
else
    echo "❌ Error clearing database"
    exit 1
fi