ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

-- Existing tokens were issued before this boundary. Setting it at migration
-- time signs all of them out once so the invalidation check is effective
-- immediately after deployment.
UPDATE users
SET password_changed_at = NOW()
WHERE password_changed_at IS NULL;
