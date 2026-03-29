CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_user_expires
  ON app.user_refresh_tokens (user_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_token_id
  ON app.user_refresh_tokens (token_id);
