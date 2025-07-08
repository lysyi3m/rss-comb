-- DOWN Migration
-- Restore redundant feed_ prefixes
ALTER TABLE feeds RENAME COLUMN url TO feed_url;
ALTER TABLE feeds RENAME COLUMN title TO feed_name;
ALTER TABLE feeds RENAME COLUMN image_url TO feed_image_url;