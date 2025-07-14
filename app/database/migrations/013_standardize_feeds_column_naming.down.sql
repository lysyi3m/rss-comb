-- DOWN Migration
-- Revert feeds table column naming changes

-- Revert boolean column name
ALTER TABLE feeds RENAME COLUMN is_enabled TO enabled;

-- Revert timestamp column names
ALTER TABLE feeds RENAME COLUMN next_fetch_at TO next_fetch;
ALTER TABLE feeds RENAME COLUMN last_fetched_at TO last_fetched;

-- Revert index to original name and column
DROP INDEX IF EXISTS idx_feeds_next_fetch_at;
CREATE INDEX idx_feeds_next_fetch ON feeds(next_fetch) WHERE enabled = true;