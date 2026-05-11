ALTER TABLE app.play_media
  ADD COLUMN IF NOT EXISTS variants jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS blurhash text NULL;

ALTER TABLE app.play_edit_suggestion_media
  ADD COLUMN IF NOT EXISTS variants jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS blurhash text NULL;

ALTER TABLE app.user_profiles
  ADD COLUMN IF NOT EXISTS avatar_variants jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS avatar_blurhash text NULL;
