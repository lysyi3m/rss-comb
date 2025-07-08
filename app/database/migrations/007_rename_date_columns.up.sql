-- UP Migration
-- Rename published_date and updated_date columns to published_at and updated_at for consistency
ALTER TABLE feed_items RENAME COLUMN published_date TO published_at;
ALTER TABLE feed_items RENAME COLUMN updated_date TO updated_at;

-- Update the index to use the new column name
DROP INDEX IF EXISTS idx_feed_items_visible;
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_at DESC)
    WHERE is_filtered = false;