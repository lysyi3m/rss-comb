-- UP Migration
-- Standardize feeds table column naming for consistency

-- Rename timestamp columns to include '_at' suffix
ALTER TABLE feeds RENAME COLUMN last_fetched TO last_fetched_at;
ALTER TABLE feeds RENAME COLUMN next_fetch TO next_fetch_at;

-- Rename boolean column to include 'is_' prefix
ALTER TABLE feeds RENAME COLUMN enabled TO is_enabled;

-- Update index to use new column name
DROP INDEX IF EXISTS idx_feeds_next_fetch;
CREATE INDEX idx_feeds_next_fetch_at ON feeds(next_fetch_at) WHERE is_enabled = true;