#!/bin/bash
# Safe table compression script that monitors disk space
# Run this instead of the SQL script to avoid running out of space

DB_USER="root"
DB_PASS="50bcT*DaeU2-19@+Q4"
DB_HOST="127.0.0.1"
DB_PORT="4100"
DB_NAME="nectar"

# Function to check free disk space in GB
check_disk_space() {
    df -BG / | tail -1 | awk '{print $4}' | sed 's/G//'
}

# Function to compress a table
compress_table() {
    local table=$1
    local free_space=$(check_disk_space)
    
    echo "================================================"
    echo "Compressing table: $table"
    echo "Current free disk space: ${free_space}GB"
    
    # Check if we have at least 50GB free space
    if [ $free_space -lt 50 ]; then
        echo "ERROR: Not enough disk space! Need at least 50GB free."
        echo "Current free: ${free_space}GB"
        exit 1
    fi
    
    # Get table size before compression
    size_before=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
        SELECT ROUND((data_length + index_length) / 1024 / 1024 / 1024, 2) 
        FROM information_schema.tables 
        WHERE table_schema = '$DB_NAME' AND table_name = '$table';")
    
    echo "Table size before: ${size_before}GB"
    
    # Compress the table
    echo "Starting compression..."
    time mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" $DB_NAME -e "
        ALTER TABLE $table ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=8;"
    
    if [ $? -eq 0 ]; then
        # Get table size after compression
        size_after=$(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
            SELECT ROUND((data_length + index_length) / 1024 / 1024 / 1024, 2) 
            FROM information_schema.tables 
            WHERE table_schema = '$DB_NAME' AND table_name = '$table';")
        
        echo "Table size after: ${size_after}GB"
        echo "Space saved: $(echo "$size_before - $size_after" | bc)GB"
        echo "Compression successful!"
    else
        echo "ERROR: Compression failed for table $table"
        exit 1
    fi
    
    echo ""
}

# Main compression process
echo "Starting Nectar table compression"
echo "This will compress all tables uniformly for consistency"
echo "=================================="

# Stop Nectar to avoid conflicts
echo "Stopping Nectar..."
pkill -f nectar

# Start with the largest tables
compress_table "ma_tx_outs"
compress_table "tx_outs"
compress_table "tx_ins"
compress_table "txes"
compress_table "blocks"
compress_table "data"
compress_table "ma_tx_mints"
compress_table "collateral_tx_ins"
compress_table "multi_assets"
compress_table "required_signers"

# Compress remaining tables
for table in $(mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -s -N -e "
    SELECT table_name 
    FROM information_schema.tables 
    WHERE table_schema = '$DB_NAME' 
    AND table_type = 'BASE TABLE'
    AND row_format != 'Compressed'
    ORDER BY data_length + index_length DESC;"); do
    compress_table "$table"
done

# Final summary
echo "================================================"
echo "COMPRESSION COMPLETE!"
echo "================================================"

mysql -h $DB_HOST -P $DB_PORT -u $DB_USER -p"$DB_PASS" -e "
    SELECT 
        COUNT(*) as total_tables,
        SUM(CASE WHEN row_format = 'Compressed' THEN 1 ELSE 0 END) as compressed_tables,
        ROUND(SUM(data_length + index_length) / 1024 / 1024 / 1024, 2) AS total_size_gb
    FROM information_schema.tables 
    WHERE table_schema = '$DB_NAME' 
    AND table_type = 'BASE TABLE';"

echo ""
echo "You can now restart Nectar"