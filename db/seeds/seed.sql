-- Butaqueando seed data entrypoint.
-- Run with psql from repository root:
--   psql "$DATABASE_URL" -f db/seeds/seed.sql

\ir 01_users.sql
\ir 02_user_profiles.sql
\ir 03_genres.sql
\ir 04_plays.sql
\ir 05_play_relations.sql
\ir 06_social.sql
