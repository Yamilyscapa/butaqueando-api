ALTER TABLE app.plays
  ADD COLUMN IF NOT EXISTS custom_genre_name text NULL,
  ADD COLUMN IF NOT EXISTS is_custom_theater boolean NOT NULL DEFAULT false;

ALTER TABLE app.play_edit_suggestions
  ADD COLUMN IF NOT EXISTS custom_genre_name text NULL,
  ADD COLUMN IF NOT EXISTS is_custom_theater boolean NOT NULL DEFAULT false;
