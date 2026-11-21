ALTER TABLE users
    ADD COLUMN IF NOT EXISTS unlock_token_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS unlock_token_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_unlock_token_hash ON users (unlock_token_hash);
