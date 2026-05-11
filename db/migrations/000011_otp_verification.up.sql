-- Switch verification + password reset tokens to 6-digit OTP semantics.
-- The hashed code space shrinks from 2^256 (random bytes) to 10^6 distinct
-- inputs, so token_hash is no longer guaranteed unique across concurrent
-- pending users. Drop the UNIQUE constraint (lookup is now keyed by user_id)
-- and add a per-row attempt counter to bound brute-force tries.

ALTER TABLE app.email_verification_tokens
  DROP CONSTRAINT IF EXISTS email_verification_tokens_token_hash_key;

ALTER TABLE app.email_verification_tokens
  ADD COLUMN IF NOT EXISTS attempts_remaining smallint NOT NULL DEFAULT 5;

ALTER TABLE app.password_reset_tokens
  DROP CONSTRAINT IF EXISTS password_reset_tokens_token_hash_key;

ALTER TABLE app.password_reset_tokens
  ADD COLUMN IF NOT EXISTS attempts_remaining smallint NOT NULL DEFAULT 5;
