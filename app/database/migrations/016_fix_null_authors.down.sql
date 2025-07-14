-- DOWN Migration
-- Revert the NULL authors fix
-- This is a no-op migration as we cannot safely revert NULL constraint fixes
-- without potentially breaking data integrity

-- Note: This migration cannot be safely reverted as it would reintroduce
-- the NULL constraint violation issue. The up migration only fixes data
-- consistency and doesn't change the schema structure.