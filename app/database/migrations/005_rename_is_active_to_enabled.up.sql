-- UP Migration: Rename is_active column to enabled for consistency
ALTER TABLE feeds RENAME COLUMN is_active TO enabled;

-- Update index to use new column name
DROP INDEX IF EXISTS idx_feeds_next_fetch;
CREATE INDEX idx_feeds_next_fetch ON feeds(next_fetch) WHERE enabled = true;