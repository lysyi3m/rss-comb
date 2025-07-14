-- UP Migration
-- Consolidate author_name and author_email into single authors field
-- Supports multiple authors as comma-separated values in format "email (name)" or "name"

-- Add new authors column as text array to support multiple authors
ALTER TABLE feed_items ADD COLUMN authors TEXT[];

-- Migrate existing author data to new format
UPDATE feed_items 
SET authors = CASE 
    WHEN author_name IS NOT NULL AND author_email IS NOT NULL THEN 
        ARRAY[author_email || ' (' || author_name || ')']
    WHEN author_name IS NOT NULL THEN 
        ARRAY[author_name]
    WHEN author_email IS NOT NULL THEN 
        ARRAY[author_email]
    ELSE 
        ARRAY[]::TEXT[]
END
WHERE author_name IS NOT NULL OR author_email IS NOT NULL;

-- Set empty array for items with no author data
UPDATE feed_items 
SET authors = ARRAY[]::TEXT[]
WHERE authors IS NULL;

-- Make authors column NOT NULL with default empty array
ALTER TABLE feed_items ALTER COLUMN authors SET NOT NULL;
ALTER TABLE feed_items ALTER COLUMN authors SET DEFAULT ARRAY[]::TEXT[];

-- Drop old author columns
ALTER TABLE feed_items DROP COLUMN author_name;
ALTER TABLE feed_items DROP COLUMN author_email;