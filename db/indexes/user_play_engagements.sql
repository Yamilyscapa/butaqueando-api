CREATE INDEX IF NOT EXISTS idx_engagements_user_kind_created
  ON app.user_play_engagements (user_id, kind, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_engagements_play_kind_created
  ON app.user_play_engagements (play_id, kind, created_at DESC);
