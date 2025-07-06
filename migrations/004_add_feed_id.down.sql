-- Remove feed_id column from feeds table
DROP INDEX IF EXISTS idx_feeds_feed_id;
ALTER TABLE feeds DROP COLUMN IF EXISTS feed_id;