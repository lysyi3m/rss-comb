ALTER TABLE feeds RENAME COLUMN title TO source_title;
ALTER TABLE feeds ADD COLUMN title TEXT;
