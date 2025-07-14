-- UP Migration
ALTER TABLE feeds RENAME COLUMN feed_updated_at TO feed_published_at;