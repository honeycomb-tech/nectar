-- Migration: Update AddressRaw field to handle larger Byron addresses
-- Byron addresses with attributes can be up to 150 bytes in raw form

ALTER TABLE tx_outs MODIFY COLUMN address_raw VARBINARY(150);