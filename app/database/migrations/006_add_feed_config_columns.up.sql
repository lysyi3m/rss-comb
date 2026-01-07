-- Add config-related columns to feeds table
ALTER TABLE feeds ADD COLUMN is_enabled BOOLEAN DEFAULT true NOT NULL;
ALTER TABLE feeds ADD COLUMN settings JSONB;
ALTER TABLE feeds ADD COLUMN filters JSONB;
ALTER TABLE feeds ADD COLUMN config_hash VARCHAR(64);

-- Create index for enabled feeds query
CREATE INDEX idx_feeds_is_enabled ON feeds(is_enabled);
