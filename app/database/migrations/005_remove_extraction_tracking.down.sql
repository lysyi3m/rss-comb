ALTER TABLE feed_items
    ADD COLUMN content_extraction_status TEXT DEFAULT 'pending',
    ADD COLUMN content_extraction_error TEXT,
    ADD COLUMN content_extracted_at TIMESTAMP,
    ADD COLUMN extraction_attempts INT DEFAULT 0;

CREATE INDEX idx_content_extraction_status ON feed_items(content_extraction_status);
CREATE INDEX idx_extraction_attempts ON feed_items(extraction_attempts)
    WHERE content_extraction_status = 'failed';
