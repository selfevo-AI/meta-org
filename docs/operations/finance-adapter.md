# Generic Finance Adapter

Meta-Org exports AI usage and project cost facts through a generic HTTP adapter. The external finance system remains the accounting source of truth.

## Export Request

Meta-Org sends `POST` requests to the adapter `endpoint_url`.

Headers:

| Header | Description |
|---|---|
| `Content-Type: application/json` | Export payload format. |
| `Idempotency-Key` | Stable batch idempotency key. |
| `X-Meta-Org-Adapter-ID` | Finance adapter UUID. |
| `X-Meta-Org-Signature` | HMAC signature when `auth_type = hmac`. |
| `Authorization: Bearer <secret>` | Bearer token when `auth_type = bearer`. |

Payload:

```json
{
  "format_version": "meta-org.finance.export.v1",
  "batch_id": "0f2a2f10-6b4f-4de0-8d66-8b78a1e5e97a",
  "adapter_id": "fb4a44d6-8b04-441d-8a76-69a8b551e917",
  "period_start": "2026-06-01",
  "period_end": "2026-06-30",
  "currency": "CNY",
  "total_amount": 123.45,
  "idempotency_key": "finance:adapter:period:currency",
  "metadata": {},
  "lines": [
    {
      "line_id": "44ac4c54-7476-4ad8-b8a7-e2d532932ccb",
      "usage_ledger_id": "5ce39e11-2487-4e7f-969e-e17a43cc0b30",
      "project_cost_entry_id": "",
      "organization_id": "",
      "department_id": "",
      "project_id": "",
      "provider_id": "",
      "model_id": "",
      "amount": 12.34,
      "currency": "CNY",
      "metadata": {}
    }
  ]
}
```

## HMAC Signature

For `auth_type = hmac`, compute:

```text
sha256=<hex HMAC-SHA256(raw_request_body, adapter_secret)>
```

The exact raw request body bytes must be signed. Do not reformat JSON before verification.

## Adapter Response

Return `2xx` for accepted requests. Optional body:

```json
{
  "external_batch_id": "FIN-2026-0001",
  "status": "accepted",
  "metadata": {
    "voucher_type": "ai_usage"
  }
}
```

Supported response statuses map to Meta-Org statuses:

| External status | Meta-Org status |
|---|---|
| `submitted`, `exported` | `exported` |
| `accepted`, `acknowledged` | `acknowledged` |
| `posted` | `posted` |
| `reconciled` | `reconciled` |
| `rejected`, `failed`, `error` | `failed` |

## Webhook Callback

Callback endpoint:

```text
POST /api/v1/finance/webhooks/{adapterID}
```

Use the same auth method as the outbound adapter.

Example:

```json
{
  "event_type": "finance.batch.posted",
  "batch_id": "0f2a2f10-6b4f-4de0-8d66-8b78a1e5e97a",
  "external_batch_id": "FIN-2026-0001",
  "status": "posted",
  "posted_amount": 123.45,
  "currency": "CNY",
  "error_message": "",
  "lines": [
    {
      "line_id": "44ac4c54-7476-4ad8-b8a7-e2d532932ccb",
      "external_line_id": "FIN-LINE-1",
      "status": "posted"
    }
  ]
}
```

## Idempotency Rules

- Batch idempotency key is stable for adapter, period, and currency.
- The adapter must treat repeated requests with the same idempotency key as the same export.
- External systems should return the same external batch ID for duplicate submissions.
- Line IDs are immutable Meta-Org UUIDs and should be stored by the finance system.

## Retry Expectations

Meta-Org retries transient outbound failures according to adapter `retry_count` and `timeout_ms`.

Retryable:

- Network timeout.
- HTTP `429`.
- HTTP `5xx`.

Not retryable without configuration or data changes:

- HTTP `400` validation failure.
- HTTP `401` or `403` auth failure.
- Business rejection from the finance system.

Repeated failures are visible in Finance Exports and the Meta-Org Inbox.
