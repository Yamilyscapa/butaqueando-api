DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'play_availability_status'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.play_availability_status AS ENUM ('in_theaters', 'archive');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'curation_status'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.curation_status AS ENUM ('pending', 'published', 'rejected');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'media_kind'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.media_kind AS ENUM ('poster', 'photo');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'review_status'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.review_status AS ENUM ('published', 'hidden');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'comment_status'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.comment_status AS ENUM ('published', 'hidden');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'engagement_kind'
      AND n.nspname = 'app'
  ) THEN
    CREATE TYPE app.engagement_kind AS ENUM ('attended', 'wishlist');
  END IF;
END
$$;
