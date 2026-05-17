-- Revert review rating back to smallint 1..5. Half-step values are rounded
-- to the nearest whole star (round-half-up via Postgres default rounding).

ALTER TABLE app.reviews
  DROP CONSTRAINT IF EXISTS reviews_rating_check;

ALTER TABLE app.reviews
  ALTER COLUMN rating TYPE smallint USING round(rating)::smallint;

ALTER TABLE app.reviews
  ADD CONSTRAINT reviews_rating_check
  CHECK (rating BETWEEN 1 AND 5);
