-- UP Migration
-- Clean up redundant feed_ prefixes and standardize terminology
-- Since we're in the feeds table, the feed_ prefix is redundant

-- Rename columns to remove redundant prefixes and use consistent terminology
ALTER TABLE feeds RENAME COLUMN feed_url TO url;
ALTER TABLE feeds RENAME COLUMN feed_name TO title;
ALTER TABLE feeds RENAME COLUMN feed_image_url TO image_url;