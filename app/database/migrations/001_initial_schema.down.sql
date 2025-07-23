-- Drop RSS Comb database schema

-- Drop indexes first
DROP INDEX IF EXISTS idx_extraction_attempts;
DROP INDEX IF EXISTS idx_content_extraction_status;
DROP INDEX IF EXISTS idx_feeds_next_fetch_at;
DROP INDEX IF EXISTS idx_feed_items_visible;
DROP INDEX IF EXISTS idx_content_hash;

-- Drop tables (cascade will handle foreign key constraints)
DROP TABLE IF EXISTS feed_items;
DROP TABLE IF EXISTS feeds;

-- Drop UUID extension if no other applications use it
-- Note: This is commented out as other applications might use the extension
-- DROP EXTENSION IF EXISTS "uuid-ossp";