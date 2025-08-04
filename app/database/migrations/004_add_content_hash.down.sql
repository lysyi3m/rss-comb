DROP INDEX IF EXISTS idx_feeds_content_hash;
ALTER TABLE feeds DROP COLUMN IF EXISTS content_hash;