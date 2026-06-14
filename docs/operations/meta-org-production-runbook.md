# Meta-Org Production Runbook

This runbook covers the first production deployment path for a single-enterprise Meta-Org instance.

## Required Environment

Backend:

| Variable | Required | Notes |
|---|---:|---|
| `DATABASE_URL` | yes | PostgreSQL URL, for example `postgres://user:pass@host:5432/meta_org?sslmode=require`. |
| `JWT_SECRET` | yes | Use a non-default high-entropy secret. |
| `MODEL_SECRET_KEY` | yes | Exactly 32 characters. Used for provider and finance adapter secret encryption. |
| `SERVER_PORT` | no | Defaults to `8080`. |
| `CORS_ORIGINS` | yes | Comma-separated frontend origins. |
| `MIGRATIONS_PATH` | yes | Usually `migrations` in containers and `../migrations` when running from `backend/`. |

Frontend:

| Variable | Required | Notes |
|---|---:|---|
| `NEXT_PUBLIC_API_URL` | yes | Browser-visible API base, for example `https://api.example.com/api/v1`. |

## Secret Generation

PowerShell examples:

```powershell
$bytes = [byte[]]::new(48)
[System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
[Convert]::ToBase64String($bytes)
```

Use the full output for `JWT_SECRET`.

For `MODEL_SECRET_KEY`, generate a 32-character value:

```powershell
$bytes = [byte[]]::new(24)
[System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
[Convert]::ToBase64String($bytes).Substring(0, 32)
```

Keep both values in the deployment secret manager. Do not commit them.

## Fresh Setup

1. Create an empty PostgreSQL database named `meta_org`.
2. Set backend environment variables, including `DATABASE_URL`, `JWT_SECRET`, `MODEL_SECRET_KEY`, and `MIGRATIONS_PATH`.
3. Start the backend. It applies SQL migrations automatically.
4. Start the frontend with `NEXT_PUBLIC_API_URL` pointing at `/api/v1`.
5. Create the first human user through the frontend registration flow.
6. Open Meta-Org Home and confirm overview and inbox data load.

## Migrating From `harness_org`

The application does not delete or overwrite the old database. Back up first.

```bash
pg_dump "$OLD_DATABASE_URL" > harness_org_backup.sql
createdb meta_org
psql "$NEW_DATABASE_URL" < harness_org_backup.sql
```

After restore, start the new backend with `DATABASE_URL` pointing to `meta_org`. Run a smoke test before allowing users back in:

```bash
cd backend
go test ./...
go build ./cmd/server
```

## Restore Procedure

1. Stop backend writers.
2. Create a fresh restore database.
3. Restore the selected dump with `psql`.
4. Point `DATABASE_URL` at the restored database.
5. Start the backend and confirm `/api/v1/health`.
6. Verify login, Meta-Org Home, Developer Tools, and Finance Exports.

## Provider Setup

1. Open Developer Tools.
2. Create providers for OpenAI, Anthropic, and Gemini.
3. Store only real provider keys in the provider form; keys are encrypted at rest and returned as masked values.
4. Create or confirm model catalog entries with pricing and currency.
5. Run provider connection tests.
6. Use the AI Assistant to run a streaming call per provider.
7. Confirm invocation logs and AI cost summary update.

Provider key rotation:

1. Open Developer Tools, Providers.
2. Select the provider.
3. Enter the new key in Key Rotation.
4. Run Test Provider.
5. Confirm only the masked key is visible.

## Finance Adapter Setup

1. Open Finance Exports, Adapters.
2. Create a Generic Finance Adapter with endpoint URL, auth type, secret, timeout, and retry count.
3. Use HMAC unless the downstream system requires bearer auth.
4. Run Test Adapter.
5. Create an export batch for the desired period.
6. Submit the batch.
7. Confirm external acknowledgement through webhook callback.
8. Review reconciliation differences.

## Streaming Troubleshooting

Check:

- Provider is active and tested.
- Model catalog entry is active.
- Browser can reach `NEXT_PUBLIC_API_URL`.
- Reverse proxy does not buffer `text/event-stream`.
- Timeouts allow long-running streaming responses.
- Invocation logs distinguish provider errors from cancelled streams.

For proxy deployments, disable response buffering for `/api/v1/ai-gateway/stream`.

## Finance Export Retry

Failed exports remain visible in Finance Exports and Meta-Org Inbox.

1. Inspect the batch failure reason.
2. Fix adapter endpoint, auth, or downstream validation.
3. Re-submit the same batch when the failure is transient.
4. Create an adjustment batch for accounting corrections after external posting.

Do not mutate exported source usage rows. Corrections should be additive.

## Operational Checks

Run before a release:

```bash
cd backend
go test ./...
go build ./cmd/server
cd ../frontend
npm run lint
npm run build
cd ..
docker compose config
```

Expected result: all commands exit successfully, and Docker Compose renders `meta_org`.
