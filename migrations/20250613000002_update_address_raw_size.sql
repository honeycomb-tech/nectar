-- Migration: Update AddressRaw field to handle larger Byron addresses
-- Some addresses in early blocks can have very large raw representations

ALTER TABLE tx_outs ADD COLUMN IF NOT EXISTS address_raw VARBINARY(2048);
ALTER TABLE tx_outs MODIFY COLUMN address_raw VARBINARY(2048);