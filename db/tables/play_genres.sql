CREATE TABLE IF NOT EXISTS app.play_genres (
  play_id uuid NOT NULL REFERENCES app.plays(id) ON DELETE CASCADE,
  genre_id uuid NOT NULL REFERENCES app.genres(id) ON DELETE RESTRICT,
  PRIMARY KEY (play_id, genre_id)
);
