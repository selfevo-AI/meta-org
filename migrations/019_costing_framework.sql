CREATE TABLE IF NOT EXISTS currencies (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    currency_type TEXT NOT NULL DEFAULT 'fiat'
        CHECK (currency_type IN ('fiat', 'virtual')),
    symbol TEXT NOT NULL DEFAULT '',
    precision_digits INT NOT NULL DEFAULT 2,
    chain_id TEXT NOT NULL DEFAULT '',
    contract_address TEXT NOT NULL DEFAULT '',
    external_source TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO currencies (code, name, currency_type, symbol, precision_digits)
VALUES
    ('CNY', 'Chinese Yuan', 'fiat', '¥', 2),
    ('USD', 'US Dollar', 'fiat', '$', 2)
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS exchange_rate_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency TEXT NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    to_currency TEXT NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    rate NUMERIC(24,12) NOT NULL CHECK (rate > 0),
    source TEXT NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual', 'external')),
    provider TEXT NOT NULL DEFAULT '',
    external_rate_id TEXT NOT NULL DEFAULT '',
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (from_currency <> to_currency)
);

CREATE INDEX IF NOT EXISTS idx_exchange_rates_pair_effective
    ON exchange_rate_versions(from_currency, to_currency, effective_from DESC);

CREATE TABLE IF NOT EXISTS cost_rate_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_type TEXT NOT NULL
        CHECK (subject_type IN ('human', 'external_human', 'agent', 'resource', 'capability', 'tool')),
    subject_id UUID,
    scope_type TEXT NOT NULL DEFAULT '',
    scope_id UUID,
    rate_type TEXT NOT NULL DEFAULT 'fixed'
        CHECK (rate_type IN ('hourly', 'daily', 'monthly', 'token', 'unit', 'fixed')),
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    base_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    base_currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    exchange_rate_version_id UUID REFERENCES exchange_rate_versions(id) ON DELETE SET NULL,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_rate_cards_subject
    ON cost_rate_cards(subject_type, subject_id, effective_from DESC);

CREATE TABLE IF NOT EXISTS cost_budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT NOT NULL
        CHECK (scope_type IN ('organization', 'department', 'requirement', 'project', 'capability', 'workflow', 'task')),
    scope_id UUID,
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    base_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    base_currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    exchange_rate_version_id UUID REFERENCES exchange_rate_versions(id) ON DELETE SET NULL,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'closed', 'cancelled')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_budgets_scope
    ON cost_budgets(scope_type, scope_id, status);

CREATE TABLE IF NOT EXISTS cost_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_type TEXT NOT NULL DEFAULT 'actual'
        CHECK (ledger_type IN ('actual', 'estimate', 'budget', 'adjustment')),
    cost_category TEXT NOT NULL
        CHECK (cost_category IN ('human', 'resource', 'agent', 'model_token', 'capability', 'tool', 'finance', 'adjustment', 'manual')),
    source_type TEXT NOT NULL DEFAULT 'manual',
    source_id UUID,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    requirement_id UUID REFERENCES requirements(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    capability_id UUID REFERENCES capabilities(id) ON DELETE SET NULL,
    actor_id UUID,
    actor_type TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    base_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    base_currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    exchange_rate_version_id UUID REFERENCES exchange_rate_versions(id) ON DELETE SET NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'posted'
        CHECK (status IN ('draft', 'posted', 'exported', 'reconciled', 'void')),
    finance_export_line_id UUID REFERENCES finance_export_lines(id) ON DELETE SET NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_ledger_scope
    ON cost_ledger_entries(project_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_cost_ledger_source
    ON cost_ledger_entries(source_type, source_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cost_ledger_unique_source_actual
    ON cost_ledger_entries(source_type, source_id, ledger_type)
    WHERE source_id IS NOT NULL AND ledger_type = 'actual';
CREATE INDEX IF NOT EXISTS idx_cost_ledger_export
    ON cost_ledger_entries(finance_export_line_id);

ALTER TABLE requirements
    ADD COLUMN IF NOT EXISTS budget_amount NUMERIC(14,2) NOT NULL DEFAULT 0;

ALTER TABLE requirements
    ADD COLUMN IF NOT EXISTS budget_currency TEXT NOT NULL DEFAULT 'CNY';

ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS budget_currency TEXT NOT NULL DEFAULT 'CNY';

ALTER TABLE finance_export_lines
    ADD COLUMN IF NOT EXISTS cost_ledger_entry_id UUID REFERENCES cost_ledger_entries(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_lines_cost_ledger
    ON finance_export_lines(cost_ledger_entry_id)
    WHERE cost_ledger_entry_id IS NOT NULL;

INSERT INTO cost_budgets (scope_type, scope_id, amount, currency, base_amount, base_currency, metadata)
SELECT 'requirement', id, budget_amount, budget_currency, budget_amount, 'CNY',
       jsonb_build_object('backfilled_from', 'requirements.budget_amount')
FROM requirements
WHERE budget_amount > 0
ON CONFLICT DO NOTHING;

INSERT INTO cost_budgets (scope_type, scope_id, amount, currency, base_amount, base_currency, metadata)
SELECT 'project', id, budget_amount, budget_currency, budget_amount, 'CNY',
       jsonb_build_object('backfilled_from', 'projects.budget_amount')
FROM projects
WHERE budget_amount > 0
ON CONFLICT DO NOTHING;
