-- Add total_collateral field to transactions table
-- This field stores the total collateral amount for Alonzo+ transactions

ALTER TABLE txes ADD COLUMN total_collateral BIGINT DEFAULT NULL;