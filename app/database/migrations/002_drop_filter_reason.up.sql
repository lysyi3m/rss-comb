-- Drop filter_reason column from feed_items table
-- This column stored filtering reasons but is no longer needed as we only track filtered status

ALTER TABLE feed_items DROP COLUMN filter_reason;