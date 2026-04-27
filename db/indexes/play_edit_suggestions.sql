CREATE INDEX IF NOT EXISTS idx_play_edit_suggestions_status_created_at
  ON app.play_edit_suggestions (status, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestions_owner_created_at
  ON app.play_edit_suggestions (created_by_user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestions_play_created_at
  ON app.play_edit_suggestions (play_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_play_edit_suggestion_media_suggestion_sort
  ON app.play_edit_suggestion_media (suggestion_id, sort_order ASC, created_at ASC);
