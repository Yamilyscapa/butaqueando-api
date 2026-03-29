CREATE OR REPLACE FUNCTION app.enforce_play_curation_transition()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.curation_status <> OLD.curation_status THEN
    IF NOT (
      (OLD.curation_status = 'pending' AND NEW.curation_status IN ('published', 'rejected'))
      OR (OLD.curation_status = 'rejected' AND NEW.curation_status = 'pending')
    ) THEN
      RAISE EXCEPTION 'invalid curation transition: % -> %', OLD.curation_status, NEW.curation_status
        USING ERRCODE = 'check_violation';
    END IF;

    IF NEW.curation_status = 'published' THEN
      IF NEW.moderated_by_user_id IS NULL THEN
        RAISE EXCEPTION 'moderated_by_user_id is required when publishing a play'
          USING ERRCODE = 'not_null_violation';
      END IF;

      NEW.moderated_at = COALESCE(NEW.moderated_at, now());
      NEW.published_at = COALESCE(NEW.published_at, now());
      NEW.rejected_reason = NULL;
    ELSIF NEW.curation_status = 'rejected' THEN
      IF NEW.moderated_by_user_id IS NULL THEN
        RAISE EXCEPTION 'moderated_by_user_id is required when rejecting a play'
          USING ERRCODE = 'not_null_violation';
      END IF;

      IF NEW.rejected_reason IS NULL OR btrim(NEW.rejected_reason) = '' THEN
        RAISE EXCEPTION 'rejected_reason is required when rejecting a play'
          USING ERRCODE = 'not_null_violation';
      END IF;

      NEW.moderated_at = COALESCE(NEW.moderated_at, now());
      NEW.published_at = NULL;
    ELSIF NEW.curation_status = 'pending' THEN
      NEW.moderated_by_user_id = NULL;
      NEW.moderated_at = NULL;
      NEW.published_at = NULL;
      NEW.rejected_reason = NULL;
    END IF;
  END IF;

  RETURN NEW;
END;
$$;
