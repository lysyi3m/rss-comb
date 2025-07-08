-- DOWN Migration
-- Rename published_at and updated_at columns back to published_date and updated_date
ALTER TABLE feed_items RENAME COLUMN published_at TO published_date;
ALTER TABLE feed_items RENAME COLUMN updated_at TO updated_date;

-- Update the index to use the old column name
DROP INDEX IF EXISTS idx_feed_items_visible;
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_date DESC)
    WHERE is_filtered = false;