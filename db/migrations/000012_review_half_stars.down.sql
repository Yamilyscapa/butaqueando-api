-- Revert review rating back to smallint 1..5. Half-step values are rounded
-- to the nearest whole star (round-half-up via Postgres default rounding).
-- The play_rating_stats view depends on reviews.rating, so it is dropped
-- and recreated around the column type change.

DROP VIEW IF EXISTS app.play_rating_stats;

ALTER TABLE app.reviews
  DROP CONSTRAINT IF EXISTS reviews_rating_check;

ALTER TABLE app.reviews
  ALTER COLUMN rating TYPE smallint USING round(rating)::smallint;

ALTER TABLE app.reviews
  ADD CONSTRAINT reviews_rating_check
  CHECK (rating BETWEEN 1 AND 5);

CREATE OR REPLACE VIEW app.play_rating_stats AS
SELECT
  r.play_id,
  COUNT(*)::int AS review_count,
  ROUND(AVG(r.rating)::numeric, 2) AS avg_rating
FROM app.reviews r
JOIN app.plays p ON p.id = r.play_id
WHERE r.status = 'published'
  AND p.curation_status = 'published'
GROUP BY r.play_id;
