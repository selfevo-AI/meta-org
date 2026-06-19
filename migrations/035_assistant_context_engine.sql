ALTER TABLE assistant_steps DROP CONSTRAINT IF EXISTS assistant_steps_step_type_check;
ALTER TABLE assistant_steps
    ADD CONSTRAINT assistant_steps_step_type_check
    CHECK (step_type IN ('llm', 'tool_call', 'tool_result', 'memory', 'approval', 'error', 'context'));

CREATE TABLE IF NOT EXISTS context_dictionary_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_level TEXT NOT NULL CHECK (scope_level IN ('saas', 'organization', 'module')),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    module_key TEXT NOT NULL DEFAULT '',
    version_key TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('json', 'yaml', 'csv', 'xlsx')),
    source_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'ai_reviewed', 'approved', 'active', 'rejected', 'archived')),
    checksum TEXT NOT NULL DEFAULT '',
    imported_by UUID REFERENCES users(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scope_level, organization_id, module_key, version_key)
);

CREATE TABLE IF NOT EXISTS context_business_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    module_key TEXT NOT NULL,
    name TEXT NOT NULL,
    scope_level TEXT NOT NULL CHECK (scope_level IN ('saas', 'organization', 'module')),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (dictionary_version_id, module_key)
);

CREATE TABLE IF NOT EXISTS context_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    domain_id UUID REFERENCES context_business_domains(id) ON DELETE CASCADE,
    entity_key TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (dictionary_version_id, entity_key)
);

CREATE TABLE IF NOT EXISTS context_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES context_entities(id) ON DELETE CASCADE,
    field_key TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    data_type TEXT NOT NULL DEFAULT 'string',
    semantic_type TEXT NOT NULL DEFAULT '',
    sensitivity_level TEXT NOT NULL DEFAULT 'normal'
        CHECK (sensitivity_level IN ('public', 'normal', 'sensitive', 'restricted')),
    base_weight DOUBLE PRECISION NOT NULL DEFAULT 1,
    is_finance_field BOOLEAN NOT NULL DEFAULT FALSE,
    is_workflow_field BOOLEAN NOT NULL DEFAULT FALSE,
    is_governance_field BOOLEAN NOT NULL DEFAULT FALSE,
    mask_strategy TEXT NOT NULL DEFAULT 'none',
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (entity_id, field_key)
);

CREATE TABLE IF NOT EXISTS context_physical_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES context_entities(id) ON DELETE CASCADE,
    field_id UUID REFERENCES context_fields(id) ON DELETE CASCADE,
    table_name TEXT NOT NULL,
    column_name TEXT NOT NULL DEFAULT '',
    join_path JSONB NOT NULL DEFAULT '[]',
    tenant_column TEXT NOT NULL DEFAULT 'organization_id',
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    module_key TEXT NOT NULL DEFAULT '',
    entity_key TEXT NOT NULL DEFAULT '',
    field_key TEXT NOT NULL DEFAULT '',
    rule_type TEXT NOT NULL CHECK (rule_type IN ('permission', 'workflow', 'finance', 'governance', 'weight', 'attention')),
    rule JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'approved', 'active', 'rejected', 'archived')),
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_change_proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    proposal_type TEXT NOT NULL DEFAULT 'dictionary_change',
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'blocked')),
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    review_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_migration_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dictionary_version_id UUID NOT NULL REFERENCES context_dictionary_versions(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    sql_up TEXT NOT NULL DEFAULT '',
    sql_down TEXT NOT NULL DEFAULT '',
    risk_level TEXT NOT NULL DEFAULT 'medium',
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'reviewed', 'executed', 'rejected')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES assistant_sessions(id) ON DELETE SET NULL,
    dictionary_version_id UUID REFERENCES context_dictionary_versions(id) ON DELETE SET NULL,
    actor_id UUID NOT NULL,
    actor_type TEXT NOT NULL DEFAULT '',
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    module_key TEXT NOT NULL DEFAULT '',
    target_type TEXT NOT NULL DEFAULT '',
    target_id UUID,
    workflow_id UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    attention_core JSONB NOT NULL DEFAULT '[]',
    supporting_context JSONB NOT NULL DEFAULT '[]',
    risk_and_signals JSONB NOT NULL DEFAULT '[]',
    omissions JSONB NOT NULL DEFAULT '[]',
    weights JSONB NOT NULL DEFAULT '{}',
    validations JSONB NOT NULL DEFAULT '{}',
    provenance JSONB NOT NULL DEFAULT '{}',
    token_budget INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_context_dictionary_versions_scope
    ON context_dictionary_versions(scope_level, organization_id, module_key, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_context_rules_lookup
    ON context_rules(module_key, entity_key, field_key, rule_type, status);
CREATE INDEX IF NOT EXISTS idx_context_packages_session
    ON context_packages(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_context_packages_target
    ON context_packages(module_key, target_type, target_id, created_at DESC);

WITH seed_version AS (
    INSERT INTO context_dictionary_versions (scope_level, module_key, version_key, source_type, source_name, status, metadata)
    VALUES ('saas', 'assistant_seed', 'assistant-context-seed-v1', 'json', 'migration_035_seed', 'active',
        '{"seed":true,"domains":["project","finance","governance"]}'::jsonb)
    ON CONFLICT (scope_level, organization_id, module_key, version_key) DO UPDATE
    SET status = 'active', updated_at = NOW()
    RETURNING id
),
domains AS (
    INSERT INTO context_business_domains (dictionary_version_id, module_key, name, scope_level, status)
    SELECT id, 'project', 'Project', 'saas', 'active' FROM seed_version
    UNION ALL SELECT id, 'finance', 'Finance', 'saas', 'active' FROM seed_version
    UNION ALL SELECT id, 'governance', 'Governance', 'saas', 'active' FROM seed_version
    ON CONFLICT (dictionary_version_id, module_key) DO NOTHING
    RETURNING id, module_key
),
entities AS (
    INSERT INTO context_entities (dictionary_version_id, domain_id, entity_key, display_name, status)
    SELECT seed_version.id, domains.id, 'project', 'Project', 'active' FROM seed_version, domains WHERE domains.module_key = 'project'
    UNION ALL SELECT seed_version.id, domains.id, 'requirement', 'Requirement', 'active' FROM seed_version, domains WHERE domains.module_key = 'project'
    UNION ALL SELECT seed_version.id, domains.id, 'cost_ledger_entry', 'Cost Ledger Entry', 'active' FROM seed_version, domains WHERE domains.module_key = 'finance'
    UNION ALL SELECT seed_version.id, domains.id, 'access_decision', 'Access Decision', 'active' FROM seed_version, domains WHERE domains.module_key = 'governance'
    ON CONFLICT (dictionary_version_id, entity_key) DO NOTHING
    RETURNING id, entity_key
)
INSERT INTO context_rules (dictionary_version_id, module_key, entity_key, field_key, rule_type, rule, status)
SELECT id, 'project', 'project', 'status', 'attention', '{"base_weight":8,"attention_core":true}'::jsonb, 'active' FROM seed_version
UNION ALL SELECT id, 'project', 'requirement', 'risk_level', 'workflow', '{"stage_multiplier":{"analysis":1.5,"execution":0.8}}'::jsonb, 'active' FROM seed_version
UNION ALL SELECT id, 'finance', 'cost_ledger_entry', 'amount', 'finance', '{"requires_validation":true,"unverified_as_signal":true}'::jsonb, 'active' FROM seed_version
UNION ALL SELECT id, 'governance', 'access_decision', 'decision', 'permission', '{"sensitivity":"restricted","explicit_rule_required":true}'::jsonb, 'active' FROM seed_version;
