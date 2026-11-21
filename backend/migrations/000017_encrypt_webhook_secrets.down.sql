COMMENT ON COLUMN webhooks.secret IS 'HMAC secret for signature verification';

-- Deliberately does not decrypt stored secrets: rollback keeps ciphertext so
-- webhook signature verification continues to work.
