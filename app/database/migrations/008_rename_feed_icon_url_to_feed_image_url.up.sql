-- UP Migration
-- Rename feed_icon_url to feed_image_url for better clarity
-- This field stores the URL from the feed's <image><url> element
ALTER TABLE feeds RENAME COLUMN feed_icon_url TO feed_image_url;