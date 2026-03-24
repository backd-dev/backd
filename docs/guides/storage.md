---
title: File Storage
weight: 7
---

backd provides optional file storage per application backed by S3-compatible object storage. Storage is opt-in — if no configuration is present in `app.yaml`, storage endpoints return `501 Not Implemented`.

## Configuration

Add a `storage` block to `app.yaml`:

```yaml
storage:
  endpoint: https://s3.amazonaws.com   # any S3-compatible endpoint
  bucket: my-app-bucket
  region: us-east-1
  access_key_id: ${S3_ACCESS_KEY}
  secret_access_key: ${S3_SECRET_KEY}
  presign_expiry: 1h                   # optional, default: 1h
```

For local development with MinIO:

```yaml
storage:
  endpoint: http://minio:9000
  bucket: backd-apps
  region: us-east-1
  access_key_id: ${MINIO_ACCESS_KEY}
  secret_access_key: ${MINIO_SECRET_KEY}
```

Credentials should use `${VAR}` references and be applied via `api secrets apply`.

## S3 Key Structure

All files are stored with a consistent key pattern:

```
<app_name>/<file_id>/<original_filename>
```

Example: `my_shop/d6vt1naf.../product-hero.jpg`

This provides app-level isolation, stable identifiers, and human-readable paths.

## Uploading Files

### HTTP Upload

```bash
curl -X POST http://localhost:8080/v1/my_app/storage/upload \
  -H "X-Publishable-Key: <key>" \
  -H "X-Session: <token>" \
  -F "file=@photo.jpg"
```

Response:
```json
{
  "data": {
    "id": "d6vt1naf...",
    "filename": "photo.jpg",
    "size": 245760,
    "secure": false
  }
}
```

### SDK Upload

```typescript
// Upload using the SDK (browser)
const file = document.querySelector('input[type=file]').files[0]
const result = await backd.storage.upload(file)
// result = { id: "...", filename: "photo.jpg", size: 245760 }
```

Files are streamed directly to S3 — the Go server never buffers the entire file in memory.

## File Columns (`__file`)

Any column whose name ends with `__file` (double underscore) is treated as a file reference. The column stores a file ID pointing to the `_files` table.

### Define in Migration

```sql
CREATE TABLE products (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    photo__file   TEXT,               -- file reference
    manual__file  TEXT,               -- another file reference
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

### Insert with File Reference

After uploading a file, store its ID in the `__file` column:

```typescript
const photo = await backd.storage.upload(photoFile)

await backd.from('products').insert({
  name: 'Widget Pro',
  photo__file: photo.id,
})
```

### Automatic Resolution

When you query a table, backd automatically resolves `__file` columns to file descriptors:

```json
{
  "data": {
    "id": "abc123",
    "name": "Widget Pro",
    "photo__file": {
      "id": "d6vt1naf...",
      "filename": "photo.jpg",
      "mime_type": "image/jpeg",
      "size": 245760,
      "secure": false,
      "url": "https://my-bucket.s3.amazonaws.com/my_app/d6vt1naf.../photo.jpg"
    }
  }
}
```

Resolution is done in a single batch query regardless of how many rows or file columns exist — no N+1 queries.

### Secure vs Public Files

- **Public files** (`secure: false`) — direct S3 URL, always accessible
- **Secure files** (`secure: true`) — presigned URL with expiry, access controlled

Secure files include an `expires_in` field (in seconds) in the file descriptor.

## Deleting Files

```bash
DELETE /v1/my_app/storage/<file_id>
```

This removes the file from both S3 and the `_files` table.

## The `_files` Table

File metadata is stored in the reserved `_files` table:

| Column | Type | Description |
|---|---|---|
| `id` | TEXT | File ID (XID) |
| `filename` | TEXT | Original filename |
| `content_type` | TEXT | MIME type (auto-detected) |
| `size_bytes` | BIGINT | File size in bytes |
| `storage_key` | TEXT | Full S3 key |
| `bucket` | TEXT | S3 bucket name |
| `secure` | BOOLEAN | Whether the file requires presigned URLs |
| `created_at` | TIMESTAMPTZ | Upload timestamp |

## Supported Backends

Any S3-compatible service works:

| Provider | Endpoint |
|---|---|
| AWS S3 | `https://s3.amazonaws.com` |
| MinIO | `http://minio:9000` |
| Cloudflare R2 | `https://<account>.r2.cloudflarestorage.com` |
| DigitalOcean Spaces | `https://<region>.digitaloceanspaces.com` |

backd uses path-style addressing (`UsePathStyle = true`) for MinIO compatibility.
