# Media Upload Flow (Railway Buckets)

This API uses private Railway Buckets for both play media and user media.

- Plays bucket: submission media (`poster`, `photo`)
- Users bucket: profile avatars

Buckets are private, so clients never receive permanent public object URLs.

## Key principles

1. Client uploads directly to bucket using pre-signed `PUT` URL.
2. API stores only `object_key` in database.
3. API serves stable media routes and redirects to short-lived pre-signed `GET` URL.

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

## Validation defaults

- allowed image types: `image/jpeg`, `image/png`, `image/webp`
- max image size configured by `S3_MAX_IMAGE_BYTES`
- upload URL TTL: `S3_UPLOAD_URL_TTL`
- download URL TTL: `S3_DOWNLOAD_URL_TTL`
