-- Restore NOT NULL on review body. Any NULL rows are backfilled to '' before
-- the constraint is re-applied.

UPDATE app.reviews SET body = '' WHERE body IS NULL;

ALTER TABLE app.reviews
  ALTER COLUMN body DROP DEFAULT;

ALTER TABLE app.reviews
  ALTER COLUMN body SET NOT NULL;
