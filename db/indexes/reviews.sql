CREATE INDEX IF NOT EXISTS idx_reviews_play_created_at
  ON app.reviews (play_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reviews_user_created_at
  ON app.reviews (user_id, created_at DESC);
