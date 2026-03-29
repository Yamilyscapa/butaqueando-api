# AGENTS.md

## Purpose

This repository contains the backend for **Butaqueando**, a theater play discovery and social rating platform.

The repository scope is API and database logic only. The primary client is a mobile app.

## Source of Truth

- Product requirements: `prd.md`
- Database schema entrypoint: `db/schema.sql`
- Runtime entrypoint: `cmd/api/main.go`
- Route composition: `internal/http/router.go`

If instructions conflict, prefer:
1. Direct user instruction
2. `prd.md`
3. This file
4. Existing code conventions

## Never Violate

1. Rating scale is strictly **1 to 5**.
2. A user can have only one review per play.
3. Engagement uniqueness is `(user_id, play_id, kind)`.
4. Follow uniqueness is `(follower_user_id, following_user_id)`.
5. Users cannot follow themselves.
6. Public feed/search/details must expose only `published` plays.
7. Rejected submissions must be edited/resubmitted on the **same play record**.
8. Curation state transitions are constrained to defined transitions.

## Current Repository Layout

- `cmd/api/main.go`: API process entrypoint.
- `internal/app/`: startup and dependency wiring.
- `internal/config/`: environment and config loading.
- `internal/database/`: DB connection and migration wiring.
- `internal/http/`: Gin router and middleware.
- `internal/modules/`: feature modules (`health`, `users`, `plays`, `follows`, `auth`).
- `internal/shared/`: shared errors, response helpers, pagination.
- `db/`: SQL schema, migrations, and smoke tests.

## Working Rules

### Core stack

- Use **Go** for backend code.
- Use **Gin** for HTTP routing and middleware.
- Use **GORM** for ORM/data access.
- Use JWT access/refresh tokens for authentication.
- Do not introduce alternative frameworks unless explicitly requested.

### Application layering

- Keep this shape for domain features:
  - `handler` for request/response mapping
  - `service` for business rules
  - `repository` for persistence access
- Keep `cmd/api/main.go` and `internal/app/bootstrap.go` thin.
- Keep route registration centralized in `internal/http/router.go`.
- Keep shared logic under `internal/shared/`.

### API conventions

- New endpoints should be versioned under `/v1`.
- Prefer resource-oriented paths and stable naming.
- List endpoints should use cursor pagination (`cursor`, `limit`).
- Errors should follow envelope: `code`, `message`, `details`, `requestId`.

### Database and schema

- Treat SQL under `db/` as source of truth.
- Keep migrations under `db/migrations/`.
- Keep PRD smoke checks under `db/smoke_tests_prd/`.
- Keep constraints aligned with product invariants.
- Do not use runtime auto-migrations as source of truth.

### Auth and security baseline

- Role model includes `user` and `admin`.
- Admin checks are mandatory for moderation routes.
- Keep auth rate limiting enabled on sensitive endpoints.
- Keep secure token lifecycle (short-lived access, rotated refresh).

### Commit standardization

- A subtask is a single testable deliverable unit (for example, one endpoint, one flow, or one migration).
- After each completed subtask, create one comprehensive commit that includes all related changes for that subtask.
- Avoid micro-commits for tiny intermediate edits; commit when the subtask is complete and coherent.
- Do not mix unrelated subtasks in the same commit.
- Commit message format is mandatory:
  `feature|chore|fix|update(<one-word-brief-resume>): <message>`
- `<one-word-brief-resume>` must be exactly one word that identifies the scope (for example: `auth`, `plays`, `follows`).

Examples:
- `feature(auth): add signup endpoint with input validation`
- `feature(auth): add signin endpoint with JWT token issuance`
- `feature(auth): add signout endpoint and refresh token revocation`

## Conventions For New Code

- Place feature code in `internal/modules/<domain>/`.
- Keep `internal/http/router.go` as composition-only:
  - define the top-level router and `/v1` group
  - wire module route registrars only
  - do not define domain endpoint handlers inline
- Each module owns its route definitions in `internal/modules/<domain>/routes.go`:
  - expose `const BasePath = "/<domain>"`
  - expose `RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies)`
  - mount only that module's handlers
- Validate input at API boundary before service execution.
- Keep business rules in services, not handlers.
- Keep persistence logic in repositories, not services.

## Validation Checklist

- Run `go test ./...` when applicable.
- Ensure route wiring resolves from `internal/http/router.go`.
- Ensure `internal/http/router.go` only composes modules (no inline domain route logic).
- Ensure each module exposes `routes.go` with `BasePath` and `RegisterRoutes(...)`.
- Ensure schema invariants still hold for SQL changes.
- If auth/moderation paths changed, verify role and visibility rules.
- Ensure each commit maps to exactly one completed subtask.
- Ensure commit messages follow: `feature|chore|fix|update(<one-word-brief-resume>): <message>`.
- Ensure commit history avoids micro-commits and mixed-scope commits.

## Known State

- Codebase is early-stage.
- The `plays` module is the central home for play business logic.
- Product decisions are locked in `prd.md`.
