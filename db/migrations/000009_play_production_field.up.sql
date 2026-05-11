ALTER TABLE app.plays
  ADD COLUMN IF NOT EXISTS production text NULL;
