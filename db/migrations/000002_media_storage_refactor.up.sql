DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'app'
      AND table_name = 'play_media'
      AND column_name = 'url'
  ) THEN
    ALTER TABLE app.play_media RENAME COLUMN url TO object_key;
  END IF;
END $$;

ALTER TABLE app.play_media
  DROP CONSTRAINT IF EXISTS play_media_play_id_url_key;

ALTER TABLE app.play_media
  DROP CONSTRAINT IF EXISTS play_media_play_id_object_key_key;

ALTER TABLE app.play_media
  ADD CONSTRAINT play_media_play_id_object_key_key UNIQUE (play_id, object_key);

ALTER TABLE app.user_profiles
  ADD COLUMN IF NOT EXISTS avatar_object_key text NULL;
