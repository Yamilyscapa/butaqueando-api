ALTER TABLE app.play_edit_suggestions
  DROP COLUMN IF EXISTS is_custom_theater,
  DROP COLUMN IF EXISTS custom_genre_name;

ALTER TABLE app.plays
  DROP COLUMN IF EXISTS is_custom_theater,
  DROP COLUMN IF EXISTS custom_genre_name;
