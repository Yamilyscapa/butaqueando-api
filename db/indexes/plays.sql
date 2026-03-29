CREATE INDEX IF NOT EXISTS idx_plays_curation_created_at
  ON app.plays (curation_status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_plays_availability
  ON app.plays (availability_status);

CREATE INDEX IF NOT EXISTS idx_plays_published_at
  ON app.plays (published_at DESC)
  WHERE curation_status = 'published';

CREATE INDEX IF NOT EXISTS idx_plays_search_filters
  ON app.plays (curation_status, availability_status, city, theater_name, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_plays_title_lower
  ON app.plays (lower(title));
