-- Migration: Expand address column to handle long Byron addresses
-- Some Byron addresses can be up to 500 characters when base58 encoded

ALTER TABLE tx_outs MODIFY COLUMN address VARCHAR(500) NOT NULL;