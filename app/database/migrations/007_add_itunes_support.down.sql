-- Remove iTunes podcast extension fields from feed_items table
ALTER TABLE feed_items DROP COLUMN itunes_image;
ALTER TABLE feed_items DROP COLUMN itunes_episode_type;
ALTER TABLE feed_items DROP COLUMN itunes_season;
ALTER TABLE feed_items DROP COLUMN itunes_episode;
ALTER TABLE feed_items DROP COLUMN itunes_duration;

-- Remove iTunes podcast extension fields from feeds table
ALTER TABLE feeds DROP COLUMN itunes_owner_email;
ALTER TABLE feeds DROP COLUMN itunes_owner_name;
ALTER TABLE feeds DROP COLUMN itunes_explicit;
ALTER TABLE feeds DROP COLUMN itunes_image;
ALTER TABLE feeds DROP COLUMN itunes_author;
