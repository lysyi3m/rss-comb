-- Remove duplicate-related columns since we no longer store duplicates
ALTER TABLE feed_items DROP COLUMN IF EXISTS is_duplicate;
ALTER TABLE feed_items DROP COLUMN IF EXISTS duplicate_of;

-- Remove the index that used is_duplicate
DROP INDEX IF EXISTS idx_feed_items_visible;

-- Create new index for visible items (non-filtered only)
CREATE INDEX idx_feed_items_visible ON feed_items(feed_id, published_date DESC) WHERE is_filtered = false;