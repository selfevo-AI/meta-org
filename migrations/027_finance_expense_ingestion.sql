-- 027_finance_expense_ingestion.sql
-- Finance expense ingestion, payables, payments, and richer financial dimensions.

ALTER TABLE finance_adapters
    ADD COLUMN IF NOT EXISTS adapter_type TEXT NOT NULL DEFAULT 'generic'
        CHECK (adapter_type IN ('generic', 'expense_api', 'file_import', 'scheduled_pull', 'payroll', 'model_billing', 'agent_billing'));

ALTER TABLE finance_adapters
    ADD COLUMN IF NOT EXISTS direction TEXT NOT NULL DEFAULT 'export'
        CHECK (direction IN ('export', 'import', 'bidirectional'));

ALTER TABLE finance_adapters
    ADD COLUMN IF NOT EXISTS field_mapping JSONB NOT NULL DEFAULT '{}';

ALTER TABLE finance_adapters
    ADD COLUMN IF NOT EXISTS pull_config JSONB NOT NULL DEFAULT '{}';

ALTER TABLE finance_adapters
    ADD COLUMN IF NOT EXISTS last_sync_at TIMESTAMPTZ;

ALTER TABLE finance_adapters
    ADD COLUMN IF NOT EXISTS last_sync_status TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS finance_import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_id UUID REFERENCES finance_adapters(id) ON DELETE SET NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('api', 'webhook', 'file', 'pull')),
    file_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'processing'
        CHECK (status IN ('processing', 'completed', 'completed_with_errors', 'failed')),
    total_records INT NOT NULL DEFAULT 0,
    processed_records INT NOT NULL DEFAULT 0,
    failed_records INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS finance_payables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payable_type TEXT NOT NULL DEFAULT 'expense'
        CHECK (payable_type IN ('expense', 'salary', 'project', 'model', 'agent', 'vendor')),
    source_type TEXT NOT NULL DEFAULT 'manual',
    source_id UUID,
    external_payable_id TEXT NOT NULL DEFAULT '',
    invoice_number TEXT NOT NULL DEFAULT '',
    vendor_id TEXT NOT NULL DEFAULT '',
    vendor_name TEXT NOT NULL DEFAULT '',
    employee_id TEXT NOT NULL DEFAULT '',
    employee_name TEXT NOT NULL DEFAULT '',
    agent_id UUID,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    account_code TEXT NOT NULL DEFAULT '',
    account_name TEXT NOT NULL DEFAULT '',
    cost_center_code TEXT NOT NULL DEFAULT '',
    cost_center_name TEXT NOT NULL DEFAULT '',
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    period_start DATE,
    period_end DATE,
    invoice_date DATE,
    due_date DATE,
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'partially_paid', 'paid', 'void')),
    paid_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_payables_external
    ON finance_payables(source_type, external_payable_id)
    WHERE external_payable_id <> '';

CREATE TABLE IF NOT EXISTS finance_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_number TEXT NOT NULL DEFAULT '',
    external_payment_id TEXT NOT NULL DEFAULT '',
    payment_method TEXT NOT NULL DEFAULT '',
    payer_account TEXT NOT NULL DEFAULT '',
    payee_account TEXT NOT NULL DEFAULT '',
    vendor_id TEXT NOT NULL DEFAULT '',
    vendor_name TEXT NOT NULL DEFAULT '',
    employee_id TEXT NOT NULL DEFAULT '',
    employee_name TEXT NOT NULL DEFAULT '',
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    paid_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'submitted', 'paid', 'failed', 'void')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_payments_external
    ON finance_payments(external_payment_id)
    WHERE external_payment_id <> '';

CREATE TABLE IF NOT EXISTS finance_payment_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES finance_payments(id) ON DELETE CASCADE,
    payable_id UUID NOT NULL REFERENCES finance_payables(id) ON DELETE CASCADE,
    amount NUMERIC(18,8) NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finance_payment_allocations_payment
    ON finance_payment_allocations(payment_id);

CREATE INDEX IF NOT EXISTS idx_finance_payment_allocations_payable
    ON finance_payment_allocations(payable_id);

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS expense_type TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS account_code TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS account_name TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS cost_center_code TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS cost_center_name TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS vendor_id TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS vendor_name TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS employee_id TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS employee_name TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS agent_id UUID;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS agent_name TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS tax_amount NUMERIC(18,8) NOT NULL DEFAULT 0;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS tax_rate NUMERIC(10,6) NOT NULL DEFAULT 0;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS invoice_number TEXT NOT NULL DEFAULT '';

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS invoice_date DATE;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS payment_status TEXT NOT NULL DEFAULT ''
        CHECK (payment_status IN ('', 'unpaid', 'partially_paid', 'paid', 'failed', 'void'));

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS payment_due_date DATE;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS period_start DATE;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS period_end DATE;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS finance_import_record_id UUID;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS finance_payable_id UUID REFERENCES finance_payables(id) ON DELETE SET NULL;

ALTER TABLE cost_ledger_entries
    ADD COLUMN IF NOT EXISTS finance_payment_id UUID REFERENCES finance_payments(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS finance_import_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES finance_import_batches(id) ON DELETE CASCADE,
    adapter_id UUID REFERENCES finance_adapters(id) ON DELETE SET NULL,
    external_record_id TEXT NOT NULL,
    expense_type TEXT NOT NULL DEFAULT '',
    raw_payload JSONB NOT NULL DEFAULT '{}',
    normalized_payload JSONB NOT NULL DEFAULT '{}',
    cost_ledger_entry_id UUID REFERENCES cost_ledger_entries(id) ON DELETE SET NULL,
    payable_id UUID REFERENCES finance_payables(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'posted'
        CHECK (status IN ('posted', 'duplicate', 'failed')),
    error_message TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_import_records_external
    ON finance_import_records(adapter_id, external_record_id)
    WHERE adapter_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_finance_import_records_batch
    ON finance_import_records(batch_id, status);

DO $$ BEGIN
    ALTER TABLE cost_ledger_entries
        ADD CONSTRAINT fk_cost_ledger_finance_import_record
        FOREIGN KEY (finance_import_record_id) REFERENCES finance_import_records(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_cost_ledger_finance_dimensions
    ON cost_ledger_entries(expense_type, account_code, cost_center_code, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_cost_ledger_salary_privacy
    ON cost_ledger_entries(project_id, employee_id, expense_type)
    WHERE expense_type = 'salary';
