CREATE INDEX IF NOT EXISTS idx_account_deletion_requests_user_requested_at
  ON app.account_deletion_requests (user_id, requested_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_account_deletion_requests_user_pending
  ON app.account_deletion_requests (user_id)
  WHERE status = 'pending';
