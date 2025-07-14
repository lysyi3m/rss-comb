-- DOWN Migration
-- Restore last_success column for rollback

ALTER TABLE feeds ADD COLUMN last_success TIMESTAMP;