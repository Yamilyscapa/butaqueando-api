# Media Upload Flow (Railway Buckets)

This API uses private S3-compatible buckets (Railway Buckets in production/dev setups) for both play media and user media.

- Plays bucket: submission media (`poster`, `photo`)
- Users bucket: profile avatars

Buckets are private, so clients never receive permanent public object URLs.

## Key principles

1. Client uploads directly to bucket using a pre-signed `PUT` URL.
2. API stores only `object_key` in the database.
3. API serves stable read routes and redirects to short-lived pre-signed `GET` URLs.
4. Optional image optimization generates width variants and blurhash metadata when enabled.

## Play media flow

### 1) Request upload URL

`POST /v1/me/submissions/plays/:playId/media/uploads`

Request:

```json
{
  "kind": "poster",
  "contentType": "image/jpeg",
  "contentLength": 245128
}
```

Response:

```json
{
  "data": {
    "objectKey": "plays/<playId>/<uuid>.jpg",
    "uploadUrl": "https://..."
  }
}
```

### 2) Upload file directly to bucket

Use `PUT uploadUrl` with the same `Content-Type` declared when requesting the URL.

### 3) Confirm and attach media record

`POST /v1/me/submissions/plays/:playId/media`

Request:

```json
{
  "kind": "poster",
  "objectKey": "plays/<playId>/<uuid>.jpg",
  "altText": "Official poster",
  "sortOrder": 0
}
```

Rules:

- only submission creator can attach media
- allowed submission status: `pending` or `rejected`
- object key must match play prefix: `plays/{playId}/...`
- object must exist in bucket (`HeadObject` check)

## Avatar flow

### 1) Request avatar upload URL

`POST /v1/me/profile/avatar/uploads`

Request:

```json
{
  "contentType": "image/png",
  "contentLength": 148220
}
```

Response:

```json
{
  "data": {
    "objectKey": "users/<userId>/avatar/<uuid>.png",
    "uploadUrl": "https://..."
  }
}
```

### 2) Upload file directly to bucket

Use `PUT uploadUrl` with the declared `Content-Type`.

### 3) Confirm avatar on profile patch

`PATCH /v1/me/profile`

- set avatar:

```json
{
  "avatarObjectKey": "users/<userId>/avatar/<uuid>.png"
}
```

- clear avatar:

```json
{
  "avatarObjectKey": ""
}
```

If `avatarObjectKey` is omitted, avatar remains unchanged.

## Stable read endpoints

- `GET /v1/media/plays/:playId/:mediaId`
- `GET /v1/media/users/:userId/avatar`

These routes return a temporary redirect to a short-lived pre-signed `GET` URL.

### Width variants (`w` query)

Both read routes accept an optional `w` query param:

- example: `GET /v1/media/plays/:playId/:mediaId?w=720`
- example: `GET /v1/media/users/:userId/avatar?w=240`

Behavior:

- invalid or missing `w` falls back to original object routing
- very large values are clamped server-side
- when optimization variants are available, the nearest configured width variant is served

Configured widths come from:

- `IMAGE_VARIANT_WIDTHS_PLAYS`
- `IMAGE_VARIANT_WIDTHS_AVATARS`

### Caching and redirect behavior

- Redirect target URLs are pre-signed and short-lived (`S3_DOWNLOAD_URL_TTL`).
- Avatar redirect responses are marked `Cache-Control: no-store, private`.
- Media resolution can use Redis-backed cache when `REDIS_URL` is configured.

## Validation defaults

- allowed image types: `image/jpeg`, `image/png`, `image/webp`
- max image size configured by `S3_MAX_IMAGE_BYTES`
- upload URL TTL: `S3_UPLOAD_URL_TTL`
- download URL TTL: `S3_DOWNLOAD_URL_TTL`

## Optimization pipeline

When `IMAGE_OPTIMIZATION_ENABLED=true`, uploaded media can be processed asynchronously:

- WebP generation quality from `IMAGE_WEBP_QUALITY`
- variant generation using play/avatar width config
- optional blurhash generation controlled by `IMAGE_BLURHASH_ENABLED`
- background processing queue size controlled by `IMAGE_WORKER_POOL_SIZE`

## Local mock media bootstrap

- Run `make db-bootstrap` to apply schema + SQL seed data + seeded media uploads.
- Run `make db-seed-media` to upload only seeded media objects.
- The command uploads deterministic Unsplash photos to the same `objectKey` values seeded in SQL.
- Seeded play media uses theater-related photos; seeded user avatars use portrait photos.
- `UNSPLASH_ACCESS_KEY` is required for the media upload step.
- If bucket env vars or `UNSPLASH_ACCESS_KEY` are missing, the media upload step is skipped without failing DB bootstrap.
