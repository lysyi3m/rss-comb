-- DOWN Migration
-- Restore separate author_name and author_email columns

-- Add back the original author columns
ALTER TABLE feed_items ADD COLUMN author_name TEXT;
ALTER TABLE feed_items ADD COLUMN author_email TEXT;

-- Extract author data from authors array back to separate columns
-- This is a best-effort migration as multiple authors will be lost
UPDATE feed_items 
SET 
    author_name = CASE 
        -- Extract name from "email (name)" format
        WHEN array_length(authors, 1) > 0 AND authors[1] ~ '\([^)]+\)$' THEN 
            trim(substring(authors[1] from '\(([^)]+)\)$'), '()')
        -- Use full string if no email format detected
        WHEN array_length(authors, 1) > 0 THEN 
            authors[1]
        ELSE 
            NULL
    END,
    author_email = CASE 
        -- Extract email from "email (name)" format
        WHEN array_length(authors, 1) > 0 AND authors[1] ~ '^[^(]+\s*\(' THEN 
            trim(substring(authors[1] from '^([^(]+)\s*\('))
        -- Check if the string looks like an email
        WHEN array_length(authors, 1) > 0 AND authors[1] ~ '@' THEN 
            authors[1]
        ELSE 
            NULL
    END
WHERE array_length(authors, 1) > 0;

-- Drop the consolidated authors column
ALTER TABLE feed_items DROP COLUMN authors;