ALTER TABLE app.password_reset_tokens
  DROP COLUMN IF EXISTS attempts_remaining;

ALTER TABLE app.password_reset_tokens
  ADD CONSTRAINT password_reset_tokens_token_hash_key UNIQUE (token_hash);

ALTER TABLE app.email_verification_tokens
  DROP COLUMN IF EXISTS attempts_remaining;

ALTER TABLE app.email_verification_tokens
  ADD CONSTRAINT email_verification_tokens_token_hash_key UNIQUE (token_hash);
