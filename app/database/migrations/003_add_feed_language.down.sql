-- Remove language field from feeds table
DROP INDEX IF EXISTS idx_feeds_language;
ALTER TABLE feeds DROP COLUMN language;