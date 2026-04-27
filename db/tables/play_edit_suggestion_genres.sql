CREATE TABLE IF NOT EXISTS app.play_edit_suggestion_genres (
  suggestion_id uuid NOT NULL REFERENCES app.play_edit_suggestions(id) ON DELETE CASCADE,
  genre_id uuid NOT NULL REFERENCES app.genres(id) ON DELETE RESTRICT,
  PRIMARY KEY (suggestion_id, genre_id)
);
