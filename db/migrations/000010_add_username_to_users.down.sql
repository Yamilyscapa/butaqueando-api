DROP INDEX IF EXISTS app.idx_users_username_lower;

ALTER TABLE app.users DROP CONSTRAINT IF EXISTS users_username_format;

ALTER TABLE app.users DROP COLUMN IF EXISTS username;
