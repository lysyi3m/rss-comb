-- DOWN Migration
-- Add raw_data column back to feed_items table
ALTER TABLE feed_items ADD COLUMN raw_data JSONB;