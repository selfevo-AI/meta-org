-- Finance accounting aliases, project settlements, receivables, receipts, and CRUD status fields.

CREATE TABLE IF NOT EXISTS finance_settlement_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_number TEXT NOT NULL DEFAULT '',
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    requirement_id UUID REFERENCES requirements(id) ON DELETE SET NULL,
    deliverable_id UUID REFERENCES deliverables(id) ON DELETE SET NULL,
    customer_id TEXT NOT NULL DEFAULT '',
    customer_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    subtotal NUMERIC(18,8) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    total_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    settlement_date DATE,
    due_date DATE,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'posted', 'void')),
    receivable_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_settlement_orders_number
    ON finance_settlement_orders(settlement_number)
    WHERE settlement_number <> '';

CREATE INDEX IF NOT EXISTS idx_finance_settlement_orders_project
    ON finance_settlement_orders(project_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS finance_settlement_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_order_id UUID NOT NULL REFERENCES finance_settlement_orders(id) ON DELETE CASCADE,
    line_type TEXT NOT NULL DEFAULT 'manual',
    source_type TEXT NOT NULL DEFAULT '',
    source_id UUID,
    deliverable_id UUID REFERENCES deliverables(id) ON DELETE SET NULL,
    description TEXT NOT NULL DEFAULT '',
    quantity NUMERIC(18,8) NOT NULL DEFAULT 1,
    unit_price NUMERIC(18,8) NOT NULL DEFAULT 0,
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    total_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finance_settlement_lines_order
    ON finance_settlement_lines(settlement_order_id);

CREATE TABLE IF NOT EXISTS finance_receivables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receivable_type TEXT NOT NULL DEFAULT 'project',
    settlement_order_id UUID REFERENCES finance_settlement_orders(id) ON DELETE SET NULL,
    source_type TEXT NOT NULL DEFAULT 'manual',
    source_id UUID,
    external_receivable_id TEXT NOT NULL DEFAULT '',
    invoice_number TEXT NOT NULL DEFAULT '',
    customer_id TEXT NOT NULL DEFAULT '',
    customer_name TEXT NOT NULL DEFAULT '',
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    requirement_id UUID REFERENCES requirements(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    account_code TEXT NOT NULL DEFAULT '',
    account_name TEXT NOT NULL DEFAULT '',
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    period_start DATE,
    period_end DATE,
    invoice_date DATE,
    due_date DATE,
    status TEXT NOT NULL DEFAULT 'unpaid'
        CHECK (status IN ('unpaid', 'partially_received', 'paid', 'void')),
    received_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_receivables_external
    ON finance_receivables(source_type, external_receivable_id)
    WHERE external_receivable_id <> '';

CREATE INDEX IF NOT EXISTS idx_finance_receivables_project
    ON finance_receivables(project_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS finance_receivable_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receivable_id UUID NOT NULL REFERENCES finance_receivables(id) ON DELETE CASCADE,
    settlement_line_id UUID REFERENCES finance_settlement_lines(id) ON DELETE SET NULL,
    line_type TEXT NOT NULL DEFAULT 'manual',
    description TEXT NOT NULL DEFAULT '',
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    total_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finance_receivable_lines_receivable
    ON finance_receivable_lines(receivable_id);

CREATE TABLE IF NOT EXISTS finance_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_number TEXT NOT NULL DEFAULT '',
    external_receipt_id TEXT NOT NULL DEFAULT '',
    payment_method TEXT NOT NULL DEFAULT '',
    payer_account TEXT NOT NULL DEFAULT '',
    receiver_account TEXT NOT NULL DEFAULT '',
    customer_id TEXT NOT NULL DEFAULT '',
    customer_name TEXT NOT NULL DEFAULT '',
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    received_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'received'
        CHECK (status IN ('draft', 'received', 'allocated', 'void')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_receipts_external
    ON finance_receipts(external_receipt_id)
    WHERE external_receipt_id <> '';

CREATE TABLE IF NOT EXISTS finance_receipt_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES finance_receipts(id) ON DELETE CASCADE,
    receivable_id UUID NOT NULL REFERENCES finance_receivables(id) ON DELETE CASCADE,
    amount NUMERIC(18,8) NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL DEFAULT 'CNY' REFERENCES currencies(code) ON DELETE RESTRICT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finance_receipt_allocations_receipt
    ON finance_receipt_allocations(receipt_id);

CREATE INDEX IF NOT EXISTS idx_finance_receipt_allocations_receivable
    ON finance_receipt_allocations(receivable_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_finance_settlement_receivable'
    ) THEN
        ALTER TABLE finance_settlement_orders
            ADD CONSTRAINT fk_finance_settlement_receivable
            FOREIGN KEY (receivable_id) REFERENCES finance_receivables(id) ON DELETE SET NULL
            NOT VALID;
    END IF;
END $$;

ALTER TABLE finance_payables
    ADD COLUMN IF NOT EXISTS void_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE finance_payments
    ADD COLUMN IF NOT EXISTS void_reason TEXT NOT NULL DEFAULT '';
