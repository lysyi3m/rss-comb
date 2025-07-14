-- UP Migration
-- Remove redundant last_success column from feeds table
-- The updated_at column serves the same purpose (tracks last successful processing)

ALTER TABLE feeds DROP COLUMN last_success;