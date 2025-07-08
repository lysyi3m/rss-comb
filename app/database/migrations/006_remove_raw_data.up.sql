-- UP Migration
-- Remove raw_data column from feed_items table
ALTER TABLE feed_items DROP COLUMN raw_data;