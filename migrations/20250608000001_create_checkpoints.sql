-- Create checkpoints table for smart resume functionality
-- This enables fast restarts without re-processing entire eras

CREATE TABLE IF NOT EXISTS sync_checkpoints (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    
    -- Blockchain position
    slot_no BIGINT UNSIGNED NOT NULL,
    block_hash VARCHAR(64) NOT NULL,
    block_id BIGINT UNSIGNED NOT NULL,
    era VARCHAR(20) NOT NULL,
    epoch_no INT UNSIGNED NOT NULL,
    
    -- Checkpoint metadata
    checkpoint_type ENUM('era_boundary', 'block_interval', 'manual') NOT NULL DEFAULT 'block_interval',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_validated BOOLEAN DEFAULT FALSE,
    
    -- Data integrity verification
    integrity_hash VARCHAR(64) NOT NULL,
    table_counts JSON NOT NULL,
    
    -- Reference points for resume
    last_tx_id BIGINT UNSIGNED NOT NULL,
    last_block_id BIGINT UNSIGNED NOT NULL,
    last_tx_out_id BIGINT UNSIGNED NOT NULL,
    
    -- Performance stats at checkpoint
    total_blocks BIGINT UNSIGNED NOT NULL,
    total_txs BIGINT UNSIGNED NOT NULL,
    processing_rate DECIMAL(10,2) DEFAULT 0.00,
    
    -- Notes for debugging
    notes TEXT,
    
    -- Indexes for fast lookup
    INDEX idx_slot_no (slot_no),
    INDEX idx_era (era),
    INDEX idx_checkpoint_type (checkpoint_type),
    INDEX idx_created_at (created_at),
    INDEX idx_validated (is_validated),
    
    -- Unique constraint to prevent duplicate checkpoints
    UNIQUE KEY unique_slot_checkpoint (slot_no, checkpoint_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- Create table for checkpoint resume attempts (for debugging)
CREATE TABLE IF NOT EXISTS checkpoint_resume_log (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    checkpoint_id BIGINT UNSIGNED NOT NULL,
    resume_attempted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resume_successful BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    recovery_time_seconds INT UNSIGNED,
    
    FOREIGN KEY (checkpoint_id) REFERENCES sync_checkpoints(id) ON DELETE CASCADE,
    INDEX idx_attempted_at (resume_attempted_at),
    INDEX idx_successful (resume_successful)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- Insert initial genesis checkpoint
INSERT INTO sync_checkpoints (
    slot_no, block_hash, block_id, era, epoch_no, 
    checkpoint_type, integrity_hash, table_counts,
    last_tx_id, last_block_id, last_tx_out_id,
    total_blocks, total_txs, notes
) VALUES (
    0, 'genesis', 0, 'byron', 0,
    'manual', 'genesis_hash', '{}',
    0, 0, 0,
    0, 0, 'Genesis checkpoint for clean starts'
) ON DUPLICATE KEY UPDATE notes = 'Genesis checkpoint for clean starts';