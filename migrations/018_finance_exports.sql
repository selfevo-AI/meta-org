CREATE TABLE IF NOT EXISTS finance_adapters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    endpoint_url TEXT NOT NULL,
    auth_type TEXT NOT NULL DEFAULT 'hmac' CHECK (auth_type IN ('hmac', 'bearer')),
    encrypted_secret TEXT NOT NULL,
    masked_secret TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error')),
    timeout_ms INT NOT NULL DEFAULT 30000,
    retry_count INT NOT NULL DEFAULT 3,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS finance_export_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_id UUID NOT NULL REFERENCES finance_adapters(id) ON DELETE RESTRICT,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'ready', 'exporting', 'exported', 'acknowledged', 'posted', 'reconciled', 'failed', 'cancelled')),
    currency TEXT NOT NULL DEFAULT 'CNY',
    total_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    external_batch_id TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL UNIQUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS finance_export_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES finance_export_batches(id) ON DELETE CASCADE,
    usage_ledger_id UUID REFERENCES ai_usage_ledger(id) ON DELETE RESTRICT,
    project_cost_entry_id UUID REFERENCES project_cost_entries(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES model_providers(id) ON DELETE SET NULL,
    model_id UUID REFERENCES models(id) ON DELETE SET NULL,
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY',
    external_line_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ready',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS finance_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_id UUID NOT NULL REFERENCES finance_adapters(id) ON DELETE RESTRICT,
    batch_id UUID REFERENCES finance_export_batches(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    signature_valid BOOLEAN NOT NULL DEFAULT FALSE,
    payload JSONB NOT NULL DEFAULT '{}',
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finance_batches_status ON finance_export_batches(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_finance_lines_batch ON finance_export_lines(batch_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_lines_usage_ledger ON finance_export_lines(usage_ledger_id) WHERE usage_ledger_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_finance_webhooks_adapter ON finance_webhook_events(adapter_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_project_cost_entries_ai_usage_source
    ON project_cost_entries(source_type, source_id)
    WHERE source_type = 'ai_usage' AND source_id IS NOT NULL;

DO $$ BEGIN
    ALTER TABLE ai_usage_ledger
        ADD CONSTRAINT fk_ai_usage_finance_export_line
        FOREIGN KEY (finance_export_line_id) REFERENCES finance_export_lines(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
