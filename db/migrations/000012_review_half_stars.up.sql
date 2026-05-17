-- Switch review rating from smallint (1..5) to numeric(2,1) with half-step
-- precision. Allowed values: 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0.
-- Existing integer rows remain valid after the type widen.
--
-- The play_rating_stats view depends on reviews.rating, so it is dropped
-- and recreated around the column type change.

DROP VIEW IF EXISTS app.play_rating_stats;

ALTER TABLE app.reviews
  DROP CONSTRAINT IF EXISTS reviews_rating_check;

ALTER TABLE app.reviews
  ALTER COLUMN rating TYPE numeric(2,1) USING rating::numeric(2,1);

ALTER TABLE app.reviews
  ADD CONSTRAINT reviews_rating_check
  CHECK (
    rating >= 1.0
    AND rating <= 5.0
    AND (rating * 2) = floor(rating * 2)
  );

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
