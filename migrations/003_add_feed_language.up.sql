-- Add language field to feeds table
ALTER TABLE feeds ADD COLUMN language VARCHAR(50) DEFAULT '';

-- Create index for language field (optional, for potential future queries)
CREATE INDEX IF NOT EXISTS idx_feeds_language ON feeds(language);