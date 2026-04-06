ALTER TABLE app.user_profiles
  DROP COLUMN IF EXISTS avatar_object_key;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'app'
      AND table_name = 'play_media'
      AND column_name = 'object_key'
  ) THEN
    ALTER TABLE app.play_media RENAME COLUMN object_key TO url;
  END IF;
END $$;

ALTER TABLE app.play_media
  DROP CONSTRAINT IF EXISTS play_media_play_id_object_key_key;

ALTER TABLE app.play_media
  DROP CONSTRAINT IF EXISTS play_media_play_id_url_key;

ALTER TABLE app.play_media
  ADD CONSTRAINT play_media_play_id_url_key UNIQUE (play_id, url);
