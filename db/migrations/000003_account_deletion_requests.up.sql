CREATE TABLE IF NOT EXISTS app.account_deletion_requests (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processed', 'rejected')),
  reason text NULL,
  requested_at timestamptz NOT NULL DEFAULT now(),
  processed_at timestamptz NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT account_deletion_requests_processed_consistency CHECK (
    (status = 'pending' AND processed_at IS NULL)
    OR (status IN ('processed', 'rejected') AND processed_at IS NOT NULL)
  )
);

CREATE INDEX IF NOT EXISTS idx_account_deletion_requests_user_requested_at
  ON app.account_deletion_requests (user_id, requested_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_account_deletion_requests_user_pending
  ON app.account_deletion_requests (user_id)
  WHERE status = 'pending';

DROP TRIGGER IF EXISTS trg_account_deletion_requests_updated_at ON app.account_deletion_requests;
CREATE TRIGGER trg_account_deletion_requests_updated_at
BEFORE UPDATE ON app.account_deletion_requests
FOR EACH ROW
EXECUTE FUNCTION app.set_updated_at();
