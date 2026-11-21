# Security Notes

## Reentrancy protection

`PaymentEscrowContract` uses a simple storage-based reentrancy guard (`DataKey::ReentrancyGuard`) around methods that perform external calls (e.g. token transfers).

Pattern:

- Read guard flag from instance storage
- Abort if already set
- Set guard `true`
- Perform state checks + external calls
- Reset guard `false` before returning

This reduces the risk of reentrant execution across external contract calls in Soroban.

## Backend authentication

- Changing or resetting a password sets `users.password_changed_at`. The JWT middleware and refresh handler reject tokens whose `iat` is at or before that boundary, so existing sessions are signed out immediately.
- Failed logins return `X-RateLimit-Remaining` plus `remaining_attempts` and `lockout_duration_seconds` in the error body. The response text states the policy without identifying whether an account exists.
- Locked accounts can be unlocked through `/api/v1/auth/request-unlock` and `/api/v1/auth/unlock`. Unlock tokens are hashed at rest, single-use, valid for one hour, and the request endpoint is rate limited.

## Webhook secrets

Webhook HMAC secrets are AES-GCM encrypted before persistence and decrypted by a GORM `AfterFind` hook. The plaintext secret is only returned once when the webhook is created and is excluded from all model JSON and structured logs.

## CORS

CORS origins are configured with `CORS_ALLOWED_ORIGINS` and must be explicit `scheme://host[:port]` values. Wildcards are rejected; credentials are only reflected for configured origins. Methods, headers, exposed headers, and preflight `Max-Age` are all configured explicitly via `CORS_*` environment variables.

