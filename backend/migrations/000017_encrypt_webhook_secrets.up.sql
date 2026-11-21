-- Encrypted webhook secrets are base64 AES-GCM payloads with a short version
-- prefix and need more room than the original 255-byte HMAC secret column.
ALTER TABLE webhooks
    ALTER COLUMN secret TYPE VARCHAR(512),
    ALTER COLUMN secret SET NOT NULL;

COMMENT ON COLUMN webhooks.secret IS 'Versioned AES-GCM ciphertext of the HMAC signing secret';
