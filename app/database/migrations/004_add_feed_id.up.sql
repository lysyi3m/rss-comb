-- Add feed_id column to feeds table for configuration feed ID
ALTER TABLE feeds ADD COLUMN feed_id TEXT;

-- Create unique index on feed_id (will be used as primary lookup)
CREATE UNIQUE INDEX idx_feeds_feed_id ON feeds(feed_id) WHERE feed_id IS NOT NULL;

-- Update existing feeds to extract feed_id from config_file name
-- This will handle the migration for existing feeds
UPDATE feeds SET feed_id = replace(replace(config_file, 'feeds/', ''), '.yml', '');

-- Make feed_id NOT NULL after populating existing records
ALTER TABLE feeds ALTER COLUMN feed_id SET NOT NULL;