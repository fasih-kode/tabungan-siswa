DROP INDEX IF EXISTS sessions_active_user_idx;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_active;
