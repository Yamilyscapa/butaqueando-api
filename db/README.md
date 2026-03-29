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

This will apply the schema and then load realistic sample data for local development.

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
