-- UP Migration
-- Fix NULL authors column constraint violation
-- This migration addresses the issue where empty author arrays were stored as NULL
-- instead of empty arrays, causing constraint violations in v0.13.1

-- Update any NULL authors to empty arrays
UPDATE feed_items 
SET authors = ARRAY[]::TEXT[]
WHERE authors IS NULL;

-- Ensure the NOT NULL constraint is properly set
-- (This should already be set from migration 015, but we ensure it here)
ALTER TABLE feed_items ALTER COLUMN authors SET NOT NULL;
ALTER TABLE feed_items ALTER COLUMN authors SET DEFAULT ARRAY[]::TEXT[];