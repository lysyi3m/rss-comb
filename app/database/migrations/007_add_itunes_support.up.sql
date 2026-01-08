-- Add iTunes podcast extension fields to feeds table
ALTER TABLE feeds ADD COLUMN itunes_author TEXT;
ALTER TABLE feeds ADD COLUMN itunes_image TEXT;
ALTER TABLE feeds ADD COLUMN itunes_explicit TEXT;
ALTER TABLE feeds ADD COLUMN itunes_owner_name TEXT;
ALTER TABLE feeds ADD COLUMN itunes_owner_email TEXT;

-- Add iTunes podcast extension fields to feed_items table
ALTER TABLE feed_items ADD COLUMN itunes_duration INTEGER;
ALTER TABLE feed_items ADD COLUMN itunes_episode INTEGER;
ALTER TABLE feed_items ADD COLUMN itunes_season INTEGER;
ALTER TABLE feed_items ADD COLUMN itunes_episode_type TEXT;
ALTER TABLE feed_items ADD COLUMN itunes_image TEXT;
