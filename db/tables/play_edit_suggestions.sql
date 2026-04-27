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
