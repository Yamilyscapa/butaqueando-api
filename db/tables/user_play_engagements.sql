CREATE TABLE IF NOT EXISTS app.user_play_engagements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  play_id uuid NOT NULL REFERENCES app.plays(id) ON DELETE CASCADE,
  kind app.engagement_kind NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, play_id, kind)
);
