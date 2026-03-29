CREATE TABLE IF NOT EXISTS app.review_comments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  review_id uuid NOT NULL REFERENCES app.reviews(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  parent_comment_id uuid NULL REFERENCES app.review_comments(id) ON DELETE CASCADE,
  body text NOT NULL,
  status app.comment_status NOT NULL DEFAULT 'published',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (parent_comment_id IS NULL OR parent_comment_id <> id)
);
