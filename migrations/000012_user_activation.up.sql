ALTER TABLE users
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX sessions_active_user_idx
    ON sessions (user_id)
    WHERE revoked_at IS NULL;
