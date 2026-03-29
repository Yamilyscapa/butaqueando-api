# Butaqueando PRD

- **Product:** Butaqueando
- **Document Type:** Product Requirements Document (PRD)
- **Scope:** Backend only (API + database)
- **Primary Client:** Mobile app (React Native + Expo)
- **Version:** v1.0
- **Status:** Approved for implementation

## 1) Product Summary

Butaqueando is a theater play discovery and social rating platform backend. It enables users to discover plays, search by relevant criteria, bookmark plays, mark plays as watched, rate from 1 to 5 stars, write reviews/comments, follow other users, and maintain a public profile with watched and rated plays.

This repository focuses on **API logic + database** only. The mobile app is the primary consumer.

## 2) Vision and Problem

### Vision

Build the go-to platform backend for theater fans to track and share their play experiences, in a Letterboxd-style model adapted to stage plays.

### Problem

Theater information and social opinion are fragmented. Users need one place to:

- Explore relevant/highlighted plays by category.
- Search quickly for specific plays.
- Save and track what they watched.
- Share ratings and opinions.

## 3) Goals (MVP)

1. Provide API + DB support for discovery feed, search, play details, ratings/reviews/comments, bookmark/watched actions, and public profiles.
2. Enable community growth by allowing any authenticated user to submit missing plays.
3. Require admin approval before a submitted play is publicly visible.
4. Optimize API behavior for mobile usage (pagination, compact payloads, stable errors, retry-safe writes).

## 4) Non-Goals (MVP)

- Ticket purchasing/checkout.
- Native app implementation in this repository.
- ML recommendations.
- External push provider integration.

## 5) User Roles

- **User**
  - Explore, search, bookmark, mark watched, rate/review/comment.
  - Follow/unfollow other users and view followings.
  - Submit missing plays.
  - View/update own profile.
- **Admin**
  - Review pending play submissions.
  - Approve/reject with moderation metadata.
  - Control curation visibility.

## 6) Core User Journeys

1. User explores highlighted and categorized feed sections.
2. User searches by title/genre/theater/city.
3. User bookmarks a play for later.
4. User marks a play as watched.
5. User rates the play (1 to 5 stars) and writes a review.
6. User comments on reviews to share thoughts.
7. User visits public profile showing watched plays and ratings.
8. User follows another user and sees their own following list.
9. User submits a new play not yet available on the platform.
10. Admin approves or rejects the submission.
11. If rejected, user edits and resubmits the **same record**.

## 7) Functional Requirements

### FR-01 Authentication and Authorization

- Authenticated identity is required for write operations.
- Role model includes `user` and `admin`.
- Admin-only routes must enforce role checks.

### FR-02 Feed and Discovery

- Feed sections include:
  - `highlighted`
  - `trending`
  - categorized lists (genre/city/theater)
- Only `published` plays are visible in public feed.

### FR-03 Search

- Search supports text query and optional filters:
  - genre
  - city
  - theater
  - availability status
- Must support cursor pagination.

### FR-04 Play Details

- Return core metadata (title, synopsis, director, duration, theater, city, genres, cast, media).
- Return aggregate stats (average rating, review count).

### FR-05 Bookmark and Watched

- Users can add/remove:
  - `wishlist` (bookmark)
  - `attended` (watched)
- Prevent duplicate engagements of the same kind per user/play.

### FR-06 Ratings and Reviews

- Rating scale is strictly **1 through 5 stars**.
- One review per user per play.
- Reviews support body text and spoiler flag.
- Users can edit their own reviews.

### FR-07 Comments

- Users can comment on reviews.
- Comments have status controls for moderation (`published` / `hidden`).

### FR-08 Public Profile

- Public profile includes:
  - bio/basic data
  - watched plays
  - ratings/reviews
  - follower/following counters

### FR-09 Community Play Submission

- Any authenticated user can submit a missing play.
- New submission is created with `curation_status = pending`.
- Pending/rejected submissions are not publicly visible.
- User can view their own submission statuses.

### FR-10 Admin Moderation

- Admin can list pending submissions.
- Admin can approve or reject.
- Reject requires a reason.

### FR-11 Resubmission Workflow (Locked)

- Rejected submission can be edited and resubmitted on the **same play record**.
- On resubmit, status transitions to `pending`.

### FR-12 Visibility Rules

- Public API responses include only `published` plays.
- Creator can access own pending/rejected submissions.
- Admin can access all statuses.

### FR-13 User Follow System

- Any authenticated user can follow another user.
- Any authenticated user can unfollow another user.
- Users cannot follow themselves.
- Duplicate follow relations are not allowed.
- Users can list their own followings with cursor pagination.

## 8) Curation State Model

- `pending -> published`
- `pending -> rejected`
- `rejected -> pending` (resubmission by creator)

## 9) API Surface (MVP)

### Public or Authenticated Read

- `GET /v1/feed?section=highlighted|trending|genre&genreId=&cursor=&limit=`
- `GET /v1/plays/:playId`
- `GET /v1/search?q=&genreId=&city=&theater=&cursor=&limit=`
- `GET /v1/plays/:playId/reviews?cursor=&limit=`
- `GET /v1/users/:userId/profile`
- `GET /v1/users/:userId/followers?cursor=&limit=`
- `GET /v1/users/:userId/followings?cursor=&limit=`

### User Mutations

- `POST /v1/plays/:playId/engagements` (`kind: wishlist|attended`)
- `DELETE /v1/plays/:playId/engagements/:kind`
- `POST /v1/plays/:playId/reviews`
- `PATCH /v1/reviews/:reviewId`
- `POST /v1/reviews/:reviewId/comments`
- `POST /v1/users/:userId/follow`
- `DELETE /v1/users/:userId/follow`
- `GET /v1/me/profile`
- `PATCH /v1/me/profile`
- `GET /v1/me/followings?cursor=&limit=`

### Submission Flow (User)

- `POST /v1/submissions/plays`
- `GET /v1/me/submissions/plays?status=`
- `PATCH /v1/me/submissions/plays/:playId` (edit/resubmit pending or rejected)

### Admin Moderation

- `GET /v1/admin/submissions/plays?status=pending&cursor=&limit=`
- `POST /v1/admin/submissions/plays/:playId/approve`
- `POST /v1/admin/submissions/plays/:playId/reject`

## 10) Database Model

The current schema draft is aligned with MVP needs:

- `app.plays`
- `app.genres`
- `app.play_genres`
- `app.play_cast_members`
- `app.play_media`
- `app.users`
- `app.user_refresh_tokens`
- `app.password_reset_tokens`
- `app.email_verification_tokens`
- `app.reviews`
- `app.review_comments`
- `app.user_play_engagements`
- `app.user_profiles`
- `app.user_follows`
- views:
  - `app.play_rating_stats`
  - `app.user_genre_stats`

### Required Constraints

- Review rating validation: `rating BETWEEN 1 AND 5`.
- One review per user per play.
- Unique user/play/engagement kind.
- Unique follower/following pair.
- Self-follow is not allowed.
- User email is unique.
- Public visibility gated by `curation_status = published`.

### Recommended Schema Additions

- Moderation metadata on plays:
  - `moderated_by_user_id` (nullable)
  - `moderated_at` (nullable)
- Search and curation indexes for mobile performance.

## 11) Relevance and Highlighting (MVP Heuristic)

Highlighted ranking score should combine:

- curation boost
- recency
- average rating
- review count
- bookmark velocity

Only `published` plays can be ranked in public highlighted feeds.

## 12) Mobile-First API Requirements

- Cursor pagination on all list endpoints.
- Stable ordering for consistent pagination.
- Compact response shape for feed cards.
- Retry-safe/idempotent behavior on write endpoints when feasible.
- Caching support (`ETag` / `If-None-Match`).
- Standardized error format: `code`, `message`, `details`, `requestId`.

## 13) Non-Functional Requirements

- Target performance (cached path): P95 feed/search/details around 300-500ms.
- Security:
  - strict route authz checks
  - validation on all inputs
  - rate limiting on sensitive endpoints
- Reliability:
  - transaction safety on multi-step writes
  - clear state transition enforcement
- Observability:
  - structured logs
  - request IDs
  - endpoint-level metrics
- Documentation:
  - OpenAPI spec for client integration

## 14) Native Go Auth and Token Lifecycle (Locked)

- Use a native Go authentication layer.
- Sign-in method for MVP: email/password.
- Require verified email for email/password sign-ins.
- Password reset flow is required.
- Role claims must support `user` and `admin`.
- Use JWT for:
  - short-lived access tokens
  - rotated refresh tokens

### Security Baseline

- `JWT_ACCESS_SECRET` with high entropy (32+ chars minimum).
- `JWT_REFRESH_SECRET` with high entropy (32+ chars minimum).
- Enable auth endpoint rate limiting.
- Keep strict token expiry and refresh rotation.
- Log security-relevant events (sign-in/session creation, token refresh, revoke).

## 15) Acceptance Criteria (MVP)

1. Authenticated users can submit missing plays; submission starts as `pending`.
2. Pending/rejected plays are excluded from public feed/search/details.
3. Admin can approve pending submission and publish it.
4. Admin can reject pending submission with a required reason.
5. Rejected submission can be edited and resubmitted on the same record.
6. User can bookmark and mark watched without duplicates.
7. User can rate (1-5) and maintain one review per play.
8. Profile endpoint returns watched plays and user ratings/reviews.
9. All list endpoints support cursor pagination.
10. API uses a standardized error response contract.
11. Users can follow/unfollow others, cannot self-follow, and can list followings without duplicates.

## 16) Success Metrics

- Percentage of new users who rate at least one play in week 1.
- Reviews per monthly active user.
- Bookmark-to-watched conversion rate.
- Follower-to-following conversion rate.
- Search success rate (query followed by result open).
- D30 retention for users with at least one review.

## 17) Milestones

### Milestone 1

- Finalize migrations and seed data.
- Implement read APIs: feed, details, search.

### Milestone 2

- Implement engagements, reviews, and comments endpoints.
- Add validation and standardized errors.

### Milestone 3

- Implement submission/moderation workflow.
- Implement profile endpoints.
- Add ranking job/cache, OpenAPI documentation, and integration tests.

## 18) Decisions Locked

- Project name: **Butaqueando**.
- This repository is backend-only.
- Primary consumer is a mobile app built with React Native + Expo.
- Rejected play submissions are edited/resubmitted on the same play record.
- Authentication stack is native Go JWT auth with refresh rotation.
