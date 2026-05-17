-- Make review body optional. Existing rows are non-null with text content;
-- after this migration the column accepts the empty string and defaults to
-- '' when omitted on insert. NULL values are treated equivalently to '' by
-- the application layer.

ALTER TABLE app.reviews
  ALTER COLUMN body DROP NOT NULL;

ALTER TABLE app.reviews
  ALTER COLUMN body SET DEFAULT '';

UPDATE app.reviews SET body = '' WHERE body IS NULL;
