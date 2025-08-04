ALTER TABLE feeds ADD COLUMN content_hash VARCHAR(16);
CREATE INDEX idx_feeds_content_hash ON feeds(content_hash);