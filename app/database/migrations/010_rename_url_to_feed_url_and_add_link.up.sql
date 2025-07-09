-- UP Migration
-- Rename feeds.url to feeds.feed_url and add feeds.link field
-- feeds.feed_url stores the RSS/Atom feed URL from configuration
-- feeds.link stores the homepage URL from the feed's <link> element (RSS 2.0 spec compliant)

-- Rename url column to feed_url to be more descriptive
ALTER TABLE feeds RENAME COLUMN url TO feed_url;

-- Add new link column to store the homepage URL from the feed
ALTER TABLE feeds ADD COLUMN link TEXT;