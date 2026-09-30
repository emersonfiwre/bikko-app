DROP INDEX IF EXISTS idx_active_users;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
