CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
  ON app.users (lower(email));

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower
  ON app.users (lower(username));
