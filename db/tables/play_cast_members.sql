CREATE TABLE IF NOT EXISTS app.play_cast_members (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  play_id uuid NOT NULL REFERENCES app.plays(id) ON DELETE CASCADE,
  person_name text NOT NULL,
  role_name text NOT NULL,
  billing_order integer NOT NULL DEFAULT 0 CHECK (billing_order >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (play_id, person_name, role_name)
);
