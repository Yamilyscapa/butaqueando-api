# Database Schema

`db/schema.sql` is the canonical schema entrypoint for this repository.

## Apply schema

```bash
psql "$DATABASE_URL" -f db/schema.sql
```

## Apply seed data

```bash
psql "$DATABASE_URL" -f db/seeds/seed.sql
```

## Bootstrap with Make

```bash
make db-bootstrap
```

This will apply the schema, load realistic sample data, and upload deterministic Unsplash media objects for local development.

If bucket variables or `UNSPLASH_ACCESS_KEY` are not configured, the media upload step is skipped and schema/data seeding still succeeds.

## Upload seeded media only

```bash
make db-seed-media
```

### Local seed accounts

- `admin@butaqueando.local` (role: `admin`)
- `ana@butaqueando.local` (role: `user`)
- `marco@butaqueando.local` (role: `user`)
- `luna@butaqueando.local` (role: `user`)
- `diego@butaqueando.local` (role: `user`)
- `carla@butaqueando.local` (role: `user`)

All seeded users share the password: `password`

## Structure

- `00_*.sql` and `01_*.sql`: setup and enum types
- `tables/`: one SQL file per table
- `seeds/`: deterministic seed data for local development
- `functions/`: trigger functions and helpers
- `triggers/`: trigger bindings
- `indexes/`: index declarations
- `views/`: derived read models
- `migrations/`: SQL migrations managed by Go tooling
- `smoke_tests_prd/`: PRD invariant smoke tests

## Media storage notes (v1)

- `app.play_media` stores `object_key` (not public URL).
- `app.play_media` uniqueness is `(play_id, object_key)`.
- `app.user_profiles.avatar_object_key` is nullable (avatar is optional).
- Read URLs for media are generated at API layer via stable endpoints + redirect to pre-signed bucket URLs.
- Seeded media uses deterministic `object_key` entries under `plays/...` and `users/...` and is backed by `make db-seed-media` uploads.
- Seeded play assets are theater-related photos; seeded user avatars are people portraits.

## Latest migration

- `000002_media_storage_refactor.up.sql`
  - renames `app.play_media.url` -> `object_key`
  - updates media uniqueness constraint
  - adds `app.user_profiles.avatar_object_key`
- `000002_media_storage_refactor.down.sql` reverts those changes.
