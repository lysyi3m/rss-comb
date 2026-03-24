DROP INDEX IF EXISTS idx_feed_items_extraction_pending;
ALTER TABLE feed_items DROP COLUMN IF EXISTS content_extraction_status;
