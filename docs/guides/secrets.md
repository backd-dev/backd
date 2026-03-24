---
title: Secrets
weight: 8
---

backd provides encrypted per-app secrets storage. Secrets are written via the CLI and read exclusively at runtime from Deno functions. No HTTP API exposes secrets.

## How Secrets Work

1. Declare secrets in `app.yaml` referencing environment variables
2. Run `api secrets apply` to encrypt and store them in the database
3. Access secrets from Deno functions via `backd.secrets.get('NAME')`

Secrets are encrypted with AES-256-GCM using per-app keys derived via HKDF from a master encryption key. The master key is never stored in the database.

## Declaring Secrets

In `app.yaml`, declare secrets with `${VAR}` references:

```yaml
secrets:
  STRIPE_KEY: ${STRIPE_KEY}
  SENDGRID_KEY: ${SENDGRID_KEY}
  WEBHOOK_SECRET: ${WEBHOOK_SECRET}
```

Rules:
- Values **must** use `${VAR_NAME}` syntax — plaintext values are rejected
- The referenced environment variable must exist and be non-empty at apply time
- Secret names are freeform strings

## Applying Secrets

```bash
# Apply secrets for all apps
api secrets apply --config-dir ./apps

# Apply for a specific app
api secrets apply --config-dir ./apps --app my_app

# Dry run — validate only, don't write
api secrets apply --config-dir ./apps --dry-run
```

### Two-Phase Safety

**Phase 1 — Validation** (no database contact): All apps are validated before any write. If any environment variable is missing, the entire command fails with a report of all missing vars.

```
Validating secrets for app: my_shop
  ✓ STRIPE_KEY
  ✓ SENDGRID_KEY
  ✗ WEBHOOK_SECRET  ← env var not set

ERROR: secrets apply failed — missing env vars
No secrets were written.
```

**Phase 2 — Apply** (one transaction per app): Each app's secrets are encrypted and written atomically. A failure in one app doesn't affect others.

## Reading Secrets in Functions

```typescript
// functions/process_payment/index.ts

export default async function handler(req: Request, backd: BackdClient) {
  const stripeKey = await backd.secrets.get('STRIPE_KEY')

  // Use the secret
  const stripe = new Stripe(stripeKey)
  // ...
}
```

Each `backd.secrets.get()` call:
1. Reads the encrypted value from `_secrets`
2. Derives the app-specific decryption key
3. Decrypts the value in memory
4. Logs the access in `_secret_audit`
5. Returns the plaintext (never cached)

## Encryption

| Component | Algorithm |
|---|---|
| Master key | 32-byte random, provided via `BACKD_ENCRYPTION_KEY` |
| Per-app key derivation | HKDF-SHA256 with fixed salt |
| Encryption | AES-256-GCM with random 12-byte nonce |
| Storage format | `[12-byte nonce][ciphertext]` |

Each encryption produces different ciphertext (random nonce), so the same secret value stored twice has different encrypted representations.

## Audit Trail

Every secret access is logged in the `_secret_audit` table:

| Column | Description |
|---|---|
| `id` | Audit record ID |
| `secret_name` | Name of the accessed secret |
| `action` | `get`, `set`, or `delete` |
| `app_name` | App context |
| `accessed_at` | Timestamp |

## Key Rotation

To rotate the master encryption key:

1. Set the new `BACKD_ENCRYPTION_KEY` environment variable
2. Re-run `api secrets apply` for all apps
3. All secrets are re-encrypted with keys derived from the new master

The HKDF salt is versioned (`backd-app-key-v1`) — changing the salt version rotates all derived keys.
