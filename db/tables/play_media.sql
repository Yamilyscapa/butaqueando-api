CREATE TABLE IF NOT EXISTS app.play_media (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  play_id uuid NOT NULL REFERENCES app.plays(id) ON DELETE CASCADE,
  kind app.media_kind NOT NULL,
  url text NOT NULL,
  alt_text text NULL,
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (play_id, url)
);
