CREATE INDEX IF NOT EXISTS idx_user_follows_follower_created
  ON app.user_follows (follower_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_follows_following_created
  ON app.user_follows (following_user_id, created_at DESC);
