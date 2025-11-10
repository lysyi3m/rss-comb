-- Remove config-related columns from feeds table
DROP INDEX IF EXISTS idx_feeds_is_enabled;
ALTER TABLE feeds DROP COLUMN IF EXISTS config_hash;
ALTER TABLE feeds DROP COLUMN IF EXISTS filters;
ALTER TABLE feeds DROP COLUMN IF EXISTS settings;
ALTER TABLE feeds DROP COLUMN IF EXISTS is_enabled;
