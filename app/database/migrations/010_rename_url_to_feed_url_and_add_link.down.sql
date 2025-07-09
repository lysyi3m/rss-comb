-- DOWN Migration
-- Revert the changes: remove link column and rename feed_url back to url

-- Remove the link column
ALTER TABLE feeds DROP COLUMN link;

-- Rename feed_url back to url
ALTER TABLE feeds RENAME COLUMN feed_url TO url;