-- Global business interaction runtime: default model matching, proposals, and internal business skills.

ALTER TABLE assistant_sessions
    ADD COLUMN IF NOT EXISTS agent_id UUID REFERENCES ai_agents(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS target_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_id UUID;

ALTER TABLE workflow_instances
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

ALTER TABLE project_evaluations
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS assistant_module_defaults (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_key TEXT NOT NULL DEFAULT 'general',
    target_type TEXT NOT NULL DEFAULT '',
    agent_id UUID REFERENCES ai_agents(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES model_providers(id) ON DELETE SET NULL,
    preferred_channel_id UUID REFERENCES model_provider_channels(id) ON DELETE SET NULL,
    provider_type TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    service_tier TEXT NOT NULL DEFAULT '',
    reasoning_effort TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (module_key, target_type)
);

CREATE INDEX IF NOT EXISTS idx_assistant_module_defaults_lookup
    ON assistant_module_defaults(module_key, target_type, updated_at DESC);

CREATE TABLE IF NOT EXISTS assistant_proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES assistant_sessions(id) ON DELETE CASCADE,
    module_key TEXT NOT NULL DEFAULT 'general',
    target_type TEXT NOT NULL DEFAULT '',
    target_id UUID,
    proposal_type TEXT NOT NULL DEFAULT 'metadata_patch',
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'applied', 'rejected')),
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    review_reason TEXT NOT NULL DEFAULT '',
    apply_result JSONB NOT NULL DEFAULT '{}',
    error_message TEXT NOT NULL DEFAULT '',
    source_step_id UUID REFERENCES assistant_steps(id) ON DELETE SET NULL,
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assistant_proposals_session
    ON assistant_proposals(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_proposals_target
    ON assistant_proposals(target_type, target_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS assistant_business_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_key TEXT NOT NULL DEFAULT 'general',
    target_type TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    trigger_intent TEXT NOT NULL DEFAULT '',
    prompt_template TEXT NOT NULL DEFAULT '',
    tool_allowlist JSONB NOT NULL DEFAULT '[]',
    input_schema JSONB NOT NULL DEFAULT '{}',
    output_schema JSONB NOT NULL DEFAULT '{}',
    version INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'archived')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by_type TEXT NOT NULL DEFAULT '',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    source_session_id UUID REFERENCES assistant_sessions(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assistant_business_skills_scope
    ON assistant_business_skills(module_key, target_type, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS assistant_skill_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id UUID NOT NULL REFERENCES assistant_business_skills(id) ON DELETE CASCADE,
    session_id UUID REFERENCES assistant_sessions(id) ON DELETE SET NULL,
    module_key TEXT NOT NULL DEFAULT 'general',
    target_type TEXT NOT NULL DEFAULT '',
    target_id UUID,
    input JSONB NOT NULL DEFAULT '{}',
    output JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'completed',
    error_message TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by_type TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_assistant_skill_runs_skill
    ON assistant_skill_runs(skill_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_skill_runs_target
    ON assistant_skill_runs(target_type, target_id, created_at DESC);

INSERT INTO assistant_module_defaults (module_key, target_type, agent_id, provider_id, provider_type, model, metadata)
SELECT module_key, target_type, agent_id, provider_id, provider_type, model_key,
    jsonb_build_object('source', 'auto_seed_first_active_model')
FROM (
    SELECT *
    FROM (VALUES
        ('meta_org', ''),
        ('requirement', 'requirement'),
        ('project', 'project'),
        ('delivery', 'deliverable'),
        ('project_cost', 'project_cost'),
        ('feedback', 'project_evaluation'),
        ('workflow', 'workflow_instance'),
        ('finance', 'finance_settlement'),
        ('finance', 'finance_receivable'),
        ('finance', 'finance_payable'),
        ('costing', 'cost_ledger_entry')
    ) AS defaults(module_key, target_type)
) defaults
CROSS JOIN LATERAL (
    SELECT m.provider_id, mp.provider_type, m.model_key
    FROM models m
    JOIN model_providers mp ON mp.id = m.provider_id
    WHERE m.status = 'active' AND mp.status = 'active'
    ORDER BY m.updated_at DESC, m.created_at DESC
    LIMIT 1
) model_defaults
LEFT JOIN LATERAL (
    SELECT id AS agent_id
    FROM ai_agents
    WHERE is_active
    ORDER BY updated_at DESC, created_at DESC
    LIMIT 1
) agent_defaults ON TRUE
ON CONFLICT (module_key, target_type) DO NOTHING;
