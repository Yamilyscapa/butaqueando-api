CREATE TABLE IF NOT EXISTS app.reviews (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  play_id uuid NOT NULL REFERENCES app.plays(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  rating numeric(2,1) NOT NULL CHECK (
    rating >= 1.0
    AND rating <= 5.0
    AND (rating * 2) = floor(rating * 2)
  ),
  title text NULL,
  body text NULL DEFAULT '',
  contains_spoilers boolean NOT NULL DEFAULT false,
  status app.review_status NOT NULL DEFAULT 'published',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (play_id, user_id)
);
