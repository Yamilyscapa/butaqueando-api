CREATE TABLE IF NOT EXISTS app.user_profiles (
  user_id uuid PRIMARY KEY REFERENCES app.users(id) ON DELETE CASCADE,
  bio text NULL,
  avatar_object_key text NULL,
  avatar_variants jsonb NOT NULL DEFAULT '[]'::jsonb,
  avatar_blurhash text NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
