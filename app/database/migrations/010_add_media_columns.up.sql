ALTER TABLE feed_items ADD COLUMN media_status TEXT;
ALTER TABLE feed_items ADD COLUMN media_path TEXT;
ALTER TABLE feed_items ADD COLUMN media_size BIGINT;

CREATE INDEX idx_feed_items_media_pending ON feed_items(id) WHERE media_status = 'pending';
CREATE INDEX idx_feed_items_media_path ON feed_items(media_path) WHERE media_status = 'ready';
