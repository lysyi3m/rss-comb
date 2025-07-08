-- DOWN Migration
-- Rename feed_image_url back to feed_icon_url
ALTER TABLE feeds RENAME COLUMN feed_image_url TO feed_icon_url;