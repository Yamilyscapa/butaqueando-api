CREATE TABLE IF NOT EXISTS app.plays (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title text NOT NULL,
  synopsis text NOT NULL,
  director text NOT NULL,
  duration_minutes integer NOT NULL CHECK (duration_minutes > 0),
  theater_name text NOT NULL,
  city text,
  availability_status app.play_availability_status NOT NULL DEFAULT 'in_theaters',
  curation_status app.curation_status NOT NULL DEFAULT 'pending',
  created_by_user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE RESTRICT,
  moderated_by_user_id uuid NULL REFERENCES app.users(id) ON DELETE RESTRICT,
  moderated_at timestamptz NULL,
  published_at timestamptz NULL,
  rejected_reason text NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT plays_published_fields_check CHECK (
    (curation_status = 'published' AND published_at IS NOT NULL)
    OR (curation_status <> 'published' AND published_at IS NULL)
  ),
  CONSTRAINT plays_rejected_reason_check CHECK (
    (curation_status = 'rejected' AND rejected_reason IS NOT NULL AND btrim(rejected_reason) <> '')
    OR (curation_status <> 'rejected' AND rejected_reason IS NULL)
  ),
  CONSTRAINT plays_moderation_fields_check CHECK (
    (curation_status = 'pending' AND moderated_by_user_id IS NULL AND moderated_at IS NULL)
    OR (curation_status IN ('published', 'rejected') AND moderated_by_user_id IS NOT NULL AND moderated_at IS NOT NULL)
  )
);
