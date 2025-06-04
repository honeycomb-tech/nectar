-- Add era column to blocks table
ALTER TABLE blocks ADD COLUMN era VARCHAR(16) NOT NULL DEFAULT 'Byron' AFTER op_cert_counter;
CREATE INDEX idx_blocks_era ON blocks(era);

-- Backfill era data based on slot boundaries (gouroboros era transition points)
-- Byron: Genesis to slot 4492799
UPDATE blocks SET era = 'Byron' WHERE slot_no < 4492800;

-- Shelley: Slot 4492800 to 16588737
UPDATE blocks SET era = 'Shelley' WHERE slot_no >= 4492800 AND slot_no < 16588738;

-- Allegra: Slot 16588738 to 23068793
UPDATE blocks SET era = 'Allegra' WHERE slot_no >= 16588738 AND slot_no < 23068794;

-- Mary: Slot 23068794 to 39916796
UPDATE blocks SET era = 'Mary' WHERE slot_no >= 23068794 AND slot_no < 39916797;

-- Alonzo: Slot 39916797 to 72316796
UPDATE blocks SET era = 'Alonzo' WHERE slot_no >= 39916797 AND slot_no < 72316797;

-- Babbage: Slot 72316797 to 133660799
UPDATE blocks SET era = 'Babbage' WHERE slot_no >= 72316797 AND slot_no < 133660800;

-- Conway: Slot 133660800 and beyond
UPDATE blocks SET era = 'Conway' WHERE slot_no >= 133660800; 