CREATE INDEX IF NOT EXISTS idx_review_comments_review_parent_created
  ON app.review_comments (review_id, parent_comment_id, created_at);
