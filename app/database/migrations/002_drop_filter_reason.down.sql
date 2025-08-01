-- Add filter_reason column back to feed_items table

ALTER TABLE feed_items ADD COLUMN filter_reason TEXT;