-- Migration: Expand address column to handle long Byron addresses
-- Some Byron addresses with full derivation paths can exceed 1000 characters
-- when CBOR encoded and then base58 encoded. This is a rare edge case from
-- early Byron era blocks with experimental features.

ALTER TABLE tx_outs MODIFY COLUMN address VARCHAR(2048) NOT NULL;