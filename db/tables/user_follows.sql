CREATE TABLE IF NOT EXISTS app.user_follows (
  follower_user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  following_user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (follower_user_id, following_user_id),
  CONSTRAINT user_follows_no_self_follow CHECK (follower_user_id <> following_user_id)
);
