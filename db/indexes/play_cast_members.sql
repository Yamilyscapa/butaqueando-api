CREATE INDEX IF NOT EXISTS idx_play_cast_members_play_billing
  ON app.play_cast_members (play_id, billing_order, created_at);
