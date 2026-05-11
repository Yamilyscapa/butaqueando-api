ALTER TABLE app.user_profiles
  DROP COLUMN IF EXISTS avatar_blurhash,
  DROP COLUMN IF EXISTS avatar_variants;

ALTER TABLE app.play_edit_suggestion_media
  DROP COLUMN IF EXISTS blurhash,
  DROP COLUMN IF EXISTS variants;

ALTER TABLE app.play_media
  DROP COLUMN IF EXISTS blurhash,
  DROP COLUMN IF EXISTS variants;
