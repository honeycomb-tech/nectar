-- Drop ALL tables in the nectar database
-- This will find and drop every table regardless of whether it's in our model definitions

SET FOREIGN_KEY_CHECKS = 0;

-- Drop all tables dynamically
SET @tables = NULL;
SELECT GROUP_CONCAT('`', table_name, '`') INTO @tables
FROM information_schema.tables
WHERE table_schema = DATABASE();

IF @tables IS NOT NULL THEN
    SET @tables = CONCAT('DROP TABLE IF EXISTS ', @tables);
    PREPARE stmt FROM @tables;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
END IF;

SET FOREIGN_KEY_CHECKS = 1;

-- Verify all tables are dropped
SELECT COUNT(*) as remaining_tables FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE();