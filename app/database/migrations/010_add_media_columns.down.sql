DROP INDEX IF EXISTS idx_feed_items_media_path;
DROP INDEX IF EXISTS idx_feed_items_media_pending;
ALTER TABLE feed_items DROP COLUMN IF EXISTS media_size;
ALTER TABLE feed_items DROP COLUMN IF EXISTS media_path;
ALTER TABLE feed_items DROP COLUMN IF EXISTS media_status;
