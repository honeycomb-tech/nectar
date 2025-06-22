-- Migration to handle extremely long Byron addresses
-- Some Byron addresses with full derivation paths can exceed 2048 characters
-- This migration increases the address column size to handle these edge cases

-- Increase address column size in tx_outs table to handle extreme Byron addresses
-- The longest observed Byron address was 26,956 characters
-- We'll use TEXT type which can handle up to 65,535 characters in MySQL/TiDB
ALTER TABLE tx_outs MODIFY COLUMN address TEXT NOT NULL;

-- Also increase address_raw column to handle larger raw address data
ALTER TABLE tx_outs MODIFY COLUMN address_raw VARBINARY(2048);

-- Note: collateral_tx_outs.address remains VARCHAR(100) as collateral addresses
-- are typically standard addresses, not the problematic Byron derivation paths