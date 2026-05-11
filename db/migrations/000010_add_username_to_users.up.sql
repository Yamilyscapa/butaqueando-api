-- Add username (@handle) to users. Backfills from email local-part with random suffix
-- so existing rows satisfy the NOT NULL + format CHECK + lower-unique index.

ALTER TABLE app.users ADD COLUMN IF NOT EXISTS username text;

UPDATE app.users
SET username = sub.candidate
FROM (
  SELECT
    id,
    -- Sanitize email local-part: lowercase, replace non-[a-z0-9_] with _.
    -- Pad too-short with "user", append md5 suffix for uniqueness, truncate to 20.
    substr(
      regexp_replace(
        lower(split_part(email, '@', 1)),
        '[^a-z0-9_]',
        '_',
        'g'
      ) || 'user' || substr(md5(id::text), 1, 6),
      1,
      20
    ) AS candidate
  FROM app.users
  WHERE username IS NULL
) AS sub
WHERE app.users.id = sub.id;

-- Any rows that still fail format (e.g. leading digit not an issue here but just in case empties)
-- get a deterministic fallback.
UPDATE app.users
SET username = 'user_' || substr(md5(id::text), 1, 14)
WHERE username !~ '^[a-z0-9_]{3,20}$';

ALTER TABLE app.users ALTER COLUMN username SET NOT NULL;

ALTER TABLE app.users
  ADD CONSTRAINT users_username_format
  CHECK (username ~ '^[a-z0-9_]{3,20}$');

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower
  ON app.users (lower(username));
