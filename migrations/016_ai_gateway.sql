CREATE TABLE IF NOT EXISTS model_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL CHECK (provider_type IN ('openai', 'anthropic', 'gemini')),
    base_url TEXT NOT NULL DEFAULT '',
    encrypted_api_key TEXT NOT NULL,
    masked_api_key TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error')),
    timeout_ms INT NOT NULL DEFAULT 60000,
    retry_count INT NOT NULL DEFAULT 1,
    risk_level TEXT NOT NULL DEFAULT 'medium' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    tags JSONB NOT NULL DEFAULT '[]',
    metadata JSONB NOT NULL DEFAULT '{}',
    last_test_status TEXT NOT NULL DEFAULT '',
    last_test_error TEXT NOT NULL DEFAULT '',
    last_tested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES model_providers(id) ON DELETE CASCADE,
    model_key TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    context_window INT NOT NULL DEFAULT 0,
    max_output_tokens INT NOT NULL DEFAULT 0,
    capabilities JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, model_key)
);

CREATE TABLE IF NOT EXISTS model_price_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    input_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    output_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY',
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES model_providers(id) ON DELETE RESTRICT,
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE RESTRICT,
    mode TEXT NOT NULL CHECK (mode IN ('sync', 'stream')),
    status TEXT NOT NULL DEFAULT 'started' CHECK (status IN ('started', 'streaming', 'completed', 'failed', 'cancelled')),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    requirement_id UUID REFERENCES requirements(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES ai_agents(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    capability_id UUID REFERENCES capabilities(id) ON DELETE SET NULL,
    source_surface TEXT NOT NULL DEFAULT '',
    request_hash TEXT NOT NULL DEFAULT '',
    provider_request_id TEXT NOT NULL DEFAULT '',
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    estimated_input_tokens INT NOT NULL DEFAULT 0,
    estimated_output_tokens INT NOT NULL DEFAULT 0,
    cost_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY',
    first_token_ms INT NOT NULL DEFAULT 0,
    duration_ms INT NOT NULL DEFAULT 0,
    error_type TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ai_usage_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invocation_id UUID NOT NULL REFERENCES ai_invocations(id) ON DELETE RESTRICT,
    model_price_version_id UUID REFERENCES model_price_versions(id) ON DELETE SET NULL,
    ledger_type TEXT NOT NULL DEFAULT 'usage' CHECK (ledger_type IN ('usage', 'adjustment')),
    amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY',
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    posted_to_project_cost BOOLEAN NOT NULL DEFAULT FALSE,
    project_cost_entry_id UUID REFERENCES project_cost_entries(id) ON DELETE SET NULL,
    finance_export_line_id UUID,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_providers_type_status ON model_providers(provider_type, status);
CREATE INDEX IF NOT EXISTS idx_models_provider_status ON models(provider_id, status);
CREATE INDEX IF NOT EXISTS idx_model_price_versions_model ON model_price_versions(model_id, effective_from DESC);
CREATE INDEX IF NOT EXISTS idx_ai_invocations_project ON ai_invocations(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_invocations_agent ON ai_invocations(agent_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_invocations_status ON ai_invocations(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_usage_ledger_invocation ON ai_usage_ledger(invocation_id);
CREATE INDEX IF NOT EXISTS idx_ai_usage_ledger_export ON ai_usage_ledger(finance_export_line_id);
