-- Butaqueando database schema entrypoint.
-- Run with psql from repository root:
--   psql "$DATABASE_URL" -f db/schema.sql

\ir 00_setup.sql
\ir 01_types.sql

\ir tables/users.sql
\ir tables/user_refresh_tokens.sql
\ir tables/password_reset_tokens.sql
\ir tables/email_verification_tokens.sql

\ir tables/plays.sql
\ir tables/genres.sql
\ir tables/play_genres.sql
\ir tables/play_cast_members.sql
\ir tables/play_media.sql
\ir tables/reviews.sql
\ir tables/review_comments.sql
\ir tables/user_play_engagements.sql
\ir tables/user_profiles.sql
\ir tables/user_follows.sql

\ir functions/set_updated_at.sql
\ir functions/enforce_play_curation_transition.sql

\ir triggers/updated_at_triggers.sql
\ir triggers/play_curation_transition_trigger.sql

\ir indexes/plays.sql
\ir indexes/play_cast_members.sql
\ir indexes/reviews.sql
\ir indexes/review_comments.sql
\ir indexes/user_play_engagements.sql
\ir indexes/user_follows.sql
\ir indexes/users.sql
\ir indexes/user_refresh_tokens.sql
\ir indexes/password_reset_tokens.sql
\ir indexes/email_verification_tokens.sql

\ir views/play_rating_stats.sql
\ir views/user_genre_stats.sql
