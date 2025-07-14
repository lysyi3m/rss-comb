-- DOWN Migration
-- Revert the malformed authors fix
-- This is a no-op migration as we cannot safely revert the data corruption fix
-- without reintroducing malformed author data like " (Name)" instead of "Name"

-- Note: This migration cannot be safely reverted as it would recreate
-- the malformed author data caused by migration 015's concatenation bug.
-- The up migration only fixes data consistency and doesn't change schema structure.