-- Drop indexes for content extraction tracking
DROP INDEX IF EXISTS idx_feed_items_extraction_attempts;
DROP INDEX IF EXISTS idx_feed_items_extraction_status;

-- Remove content extraction tracking fields from feed_items table
ALTER TABLE feed_items 
DROP COLUMN IF EXISTS extraction_attempts,
DROP COLUMN IF EXISTS content_extraction_error,
DROP COLUMN IF EXISTS content_extraction_status,
DROP COLUMN IF EXISTS content_extracted_at;