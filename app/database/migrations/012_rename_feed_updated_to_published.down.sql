-- DOWN Migration
ALTER TABLE feeds RENAME COLUMN feed_published_at TO feed_updated_at;