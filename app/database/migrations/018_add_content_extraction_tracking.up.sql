-- Add content extraction tracking fields to feed_items table
ALTER TABLE feed_items 
ADD COLUMN content_extracted_at TIMESTAMP,
ADD COLUMN content_extraction_status TEXT DEFAULT 'pending',
ADD COLUMN content_extraction_error TEXT,
ADD COLUMN extraction_attempts INTEGER DEFAULT 0;

-- Create index on extraction status for efficient querying of pending items
CREATE INDEX idx_feed_items_extraction_status ON feed_items(content_extraction_status);

-- Create index on extraction attempts for retry logic
CREATE INDEX idx_feed_items_extraction_attempts ON feed_items(extraction_attempts);