CREATE INDEX IF NOT EXISTS idx_play_genres_genre_play
  ON app.play_genres (genre_id, play_id);
