CREATE TABLE IF NOT EXISTS app.play_edit_suggestion_media (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  suggestion_id uuid NOT NULL REFERENCES app.play_edit_suggestions(id) ON DELETE CASCADE,
  kind app.media_kind NOT NULL,
  object_key text NOT NULL,
  alt_text text NULL,
  sort_order integer NOT NULL DEFAULT 0,
  variants jsonb NOT NULL DEFAULT '[]'::jsonb,
  blurhash text NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (suggestion_id, object_key)
);
