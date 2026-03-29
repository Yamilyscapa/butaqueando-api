CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_expires
  ON app.password_reset_tokens (user_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_hash
  ON app.password_reset_tokens (token_hash);
