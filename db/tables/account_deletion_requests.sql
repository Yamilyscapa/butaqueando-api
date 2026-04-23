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
