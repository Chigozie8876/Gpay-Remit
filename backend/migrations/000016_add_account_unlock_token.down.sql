DROP INDEX IF EXISTS idx_users_unlock_token_hash;

ALTER TABLE users
    DROP COLUMN IF EXISTS unlock_token_expires_at,
    DROP COLUMN IF EXISTS unlock_token_hash;
