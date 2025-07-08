-- Restore duplicate-related columns (for rollback)
ALTER TABLE feed_items ADD COLUMN IF NOT EXISTS is_duplicate BOOLEAN DEFAULT false;
ALTER TABLE feed_items ADD COLUMN IF NOT EXISTS duplicate_of UUID;

-- Restore the original index
DROP INDEX IF EXISTS idx_feed_items_visible;
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_date DESC) WHERE is_duplicate = false AND is_filtered = false;