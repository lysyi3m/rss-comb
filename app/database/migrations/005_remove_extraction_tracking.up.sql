DROP INDEX IF EXISTS idx_content_extraction_status;
DROP INDEX IF EXISTS idx_extraction_attempts;

ALTER TABLE feed_items
    DROP COLUMN IF EXISTS content_extraction_status,
    DROP COLUMN IF EXISTS content_extraction_error,
    DROP COLUMN IF EXISTS content_extracted_at,
    DROP COLUMN IF EXISTS extraction_attempts;
