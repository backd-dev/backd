---
title: Error Codes
weight: 3
---

All errors returned by the backd API follow a consistent envelope:

```json
{
  "error": "ERROR_CODE",
  "error_detail": "Human-readable description"
}
```

## Error Code Reference

### Authentication Errors (4xx)

| Code | HTTP Status | Description |
|---|---|---|
| `UNAUTHORIZED` | 401 | Invalid credentials or missing authentication |
| `SESSION_EXPIRED` | 401 | Session token has expired |
| `TOO_MANY_REQUESTS` | 429 | Rate limit exceeded (too many login attempts) |

### Authorization Errors

| Code | HTTP Status | Description |
|---|---|---|
| `FORBIDDEN` | 403 | RLS policy denied access, or authentication required by policy |

### Request Errors

| Code | HTTP Status | Description |
|---|---|---|
| `BAD_REQUEST` | 400 | Generic bad request (see `error_detail` for specifics) |
| `INVALID_JSON` | 400 | Request body is not valid JSON |
| `MISSING_FIELDS` | 400 | Required fields not provided |
| `USER_EXISTS` | 400 | Username already registered |
| `INVALID_COLLECTION` | 400 | Table name contains invalid characters |
| `INVALID_WHERE` | 400 | `where` filter is malformed |
| `INVALID_LIMIT` | 400 | Limit is not a valid non-negative integer |
| `LIMIT_TOO_HIGH` | 400 | Limit exceeds maximum (1000) |
| `INVALID_OFFSET` | 400 | Offset is not a valid non-negative integer |
| `EMPTY_PAYLOAD` | 400 | Update request has no fields to update |
| `MISSING_FILE` | 400 | No file provided in multipart upload |
| `INVALID_FORM` | 400 | Failed to parse multipart form |
| `MISSING_TOKEN` | 400 | Session token not provided |
| `MISSING_FUNCTION_NAME` | 400 | Function name not provided |
| `MISSING_JOB_ID` | 400 | Job ID not provided |
| `USERNAME_TAKEN` | 400 | New username already in use |

### Not Found Errors

| Code | HTTP Status | Description |
|---|---|---|
| `NOT_FOUND` | 404 | Row not found (or excluded by RLS policy) |
| `FUNCTION_NOT_FOUND` | 404 | Function does not exist or is private (`_` prefix) |

### Validation Errors

| Code | HTTP Status | Description |
|---|---|---|
| `VALIDATION_ERROR` | 422 | Request data failed validation |

### Server Errors

| Code | HTTP Status | Description |
|---|---|---|
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `STORAGE_DISABLED` | 501 | Storage not configured for this app |
| `SERVICE_UNAVAILABLE` | 503 | App failed to start (migration error, policy error, etc.) |
| `FUNCTION_TIMEOUT` | 504 | Function execution exceeded timeout |
| `FUNCTION_ERROR` | varies | Error returned by the function itself |

## SDK Error Types

The SDKs map error codes to typed exceptions:

| SDK Type | Matches Codes |
|---|---|
| `AuthError` | `UNAUTHORIZED`, `SESSION_EXPIRED`, `TOO_MANY_REQUESTS` |
| `QueryError` | `FORBIDDEN`, `NOT_FOUND`, `VALIDATION_ERROR`, and any 403/404/422 |
| `FunctionError` | `FUNCTION_NOT_FOUND`, `FUNCTION_TIMEOUT`, `FUNCTION_ERROR` |
| `NetworkError` | `NETWORK_ERROR` (transport-level failures) |
| `BackdError` | Base type for all other errors |

## Error Handling Best Practices

### Client-side

```typescript
try {
  await backd.from('orders').insert({ total: 99.99 })
} catch (err) {
  if (err instanceof AuthError) {
    // Redirect to login
  } else if (err instanceof QueryError && err.code === 'FORBIDDEN') {
    // Show permission denied message
  } else {
    // Generic error handling
    console.error(err.code, err.detail)
  }
}
```

### Distinguishing "not found" from "access denied"

backd intentionally returns `404 NOT_FOUND` for rows that exist but are excluded by RLS. This prevents information leakage — the client cannot distinguish between "row doesn't exist" and "row exists but you can't see it."
