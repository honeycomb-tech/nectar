#!/bin/bash

# Fix database tables for Nectar after refactor
# This script drops all tables and allows them to be recreated with proper AUTO_RANDOM support

echo "Fixing Nectar database tables..."

# Database connection parameters
DB_HOST="127.0.0.1"
DB_PORT="4000"
DB_USER="root"
DB_PASS="Ds*uS+pG278T@-3K60"
DB_NAME="nectar"

# Drop all tables
echo "Dropping all tables..."
mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" $DB_NAME < drop_all_tables.sql

if [ $? -eq 0 ]; then
    echo "✓ Tables dropped successfully"
    echo ""
    echo "Now you can start Nectar and it will create tables with proper AUTO_RANDOM support:"
    echo "  ./nectar"
    echo ""
    echo "The following tables will be created with AUTO_RANDOM:"
    echo "  - slot_leaders, blocks, epoches, txes"
    echo "  - tx_outs, tx_ins, ma_tx_outs"  
    echo "  - voting_procedures, stake_addresses, multi_assets"
    echo "  - delegations, rewards, pool_updates"
    echo "  - pool_hashes, scripts, data"
else
    echo "✗ Error dropping tables"
    exit 1
fi