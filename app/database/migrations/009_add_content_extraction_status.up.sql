ALTER TABLE feed_items ADD COLUMN content_extraction_status TEXT;

CREATE INDEX idx_feed_items_extraction_pending ON feed_items(id) WHERE content_extraction_status = 'pending';
