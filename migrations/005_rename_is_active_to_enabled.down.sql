-- DOWN Migration: Rename enabled column back to is_active
ALTER TABLE feeds RENAME COLUMN enabled TO is_active;

-- Update index to use old column name
DROP INDEX IF EXISTS idx_feeds_next_fetch;
CREATE INDEX idx_feeds_next_fetch ON feeds(next_fetch) WHERE is_active = true;