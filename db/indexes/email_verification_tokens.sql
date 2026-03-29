CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_expires
  ON app.email_verification_tokens (user_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_hash
  ON app.email_verification_tokens (token_hash);
