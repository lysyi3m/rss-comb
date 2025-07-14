-- UP Migration
-- Fix malformed author data created by migration 015
-- Corrects cases where empty emails created authors like " (Name)" instead of just "Name"

-- Fix authors that start with " (" (space + parenthesis) - these are malformed
UPDATE feed_items 
SET authors = ARRAY[
    CASE 
        WHEN authors[1] LIKE ' (%)' THEN 
            -- Extract name from " (Name)" format and return just the name
            SUBSTRING(authors[1] FROM 3 FOR LENGTH(authors[1]) - 3)
        ELSE 
            authors[1]
    END
]
WHERE array_length(authors, 1) = 1 
  AND authors[1] LIKE ' (%)';