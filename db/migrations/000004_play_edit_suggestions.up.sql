DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'play_edit_suggestion_status'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.play_edit_suggestion_status AS ENUM ('pending', 'approved', 'rejected');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS app.play_edit_suggestions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  play_id uuid NOT NULL REFERENCES app.plays(id) ON DELETE CASCADE,
  created_by_user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE RESTRICT,
  status app.play_edit_suggestion_status NOT NULL DEFAULT 'pending',
  title text NOT NULL,
  synopsis text NOT NULL,
  director text NOT NULL,
  duration_minutes integer NOT NULL CHECK (duration_minutes > 0),
  theater_name text NOT NULL,
  city text,
  availability_status app.play_availability_status NOT NULL DEFAULT 'in_theaters',
  moderated_by_user_id uuid NULL REFERENCES app.users(id) ON DELETE RESTRICT,
  moderated_at timestamptz NULL,
  rejected_reason text NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT play_edit_suggestions_moderation_check CHECK (
    (status = 'pending' AND moderated_by_user_id IS NULL AND moderated_at IS NULL AND rejected_reason IS NULL)
    OR (status = 'approved' AND moderated_by_user_id IS NOT NULL AND moderated_at IS NOT NULL AND rejected_reason IS NULL)
    OR (status = 'rejected' AND moderated_by_user_id IS NOT NULL AND moderated_at IS NOT NULL AND rejected_reason IS NOT NULL AND btrim(rejected_reason) <> '')
  )
);

CREATE TABLE IF NOT EXISTS app.play_edit_suggestion_genres (
  suggestion_id uuid NOT NULL REFERENCES app.play_edit_suggestions(id) ON DELETE CASCADE,
  genre_id uuid NOT NULL REFERENCES app.genres(id) ON DELETE RESTRICT,
  PRIMARY KEY (suggestion_id, genre_id)
);

CREATE TABLE IF NOT EXISTS app.play_edit_suggestion_media (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  suggestion_id uuid NOT NULL REFERENCES app.play_edit_suggestions(id) ON DELETE CASCADE,
  kind app.media_kind NOT NULL,
  object_key text NOT NULL,
  alt_text text NULL,
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (suggestion_id, object_key)
);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestions_status_created_at
  ON app.play_edit_suggestions (status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestions_owner_created_at
  ON app.play_edit_suggestions (created_by_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestions_play_created_at
  ON app.play_edit_suggestions (play_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestion_media_suggestion_sort
  ON app.play_edit_suggestion_media (suggestion_id, sort_order ASC, created_at ASC);

DROP TRIGGER IF EXISTS trg_play_edit_suggestions_updated_at ON app.play_edit_suggestions;
CREATE TRIGGER trg_play_edit_suggestions_updated_at
BEFORE UPDATE ON app.play_edit_suggestions
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();
