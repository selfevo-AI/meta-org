-- 012_policy_weight_evaluation.sql

ALTER TABLE ai_agents
    ADD COLUMN IF NOT EXISTS agent_origin TEXT NOT NULL DEFAULT 'internal'
        CHECK (agent_origin IN ('internal', 'external')),
    ADD COLUMN IF NOT EXISTS provider TEXT,
    ADD COLUMN IF NOT EXISTS service_class TEXT NOT NULL DEFAULT 'model',
    ADD COLUMN IF NOT EXISTS vendor TEXT,
    ADD COLUMN IF NOT EXISTS contract_ref TEXT,
    ADD COLUMN IF NOT EXISTS risk_level TEXT NOT NULL DEFAULT 'medium'
        CHECK (risk_level IN ('low', 'medium', 'high', 'critical'));

CREATE TABLE IF NOT EXISTS employee_profiles (
    user_id             UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    employee_no         TEXT UNIQUE,
    employment_type     TEXT NOT NULL DEFAULT 'internal'
        CHECK (employment_type IN ('internal', 'contractor', 'partner')),
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    title               TEXT,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS access_decisions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id            UUID NOT NULL,
    actor_type          TEXT NOT NULL,
    action              TEXT NOT NULL,
    resource            TEXT NOT NULL,
    resource_id         UUID,
    organization_id     UUID,
    department_id       UUID,
    workflow_id         UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id             UUID REFERENCES tasks(id) ON DELETE SET NULL,
    capability_id       UUID REFERENCES capabilities(id) ON DELETE SET NULL,
    required_level      TEXT NOT NULL DEFAULT 'L1',
    risk_level          TEXT NOT NULL DEFAULT 'low'
        CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    decision            TEXT NOT NULL
        CHECK (decision IN ('allow', 'notify', 'approve', 'deny')),
    allowed             BOOLEAN NOT NULL,
    behavior            TEXT NOT NULL,
    reason              TEXT NOT NULL,
    matched_rules       JSONB NOT NULL DEFAULT '[]',
    weight_snapshot     DOUBLE PRECISION,
    context             JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_weight_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id              UUID NOT NULL,
    actor_type            TEXT NOT NULL,
    scope_hash            TEXT NOT NULL,
    organization_id       UUID,
    department_id         UUID,
    workflow_template_id  UUID REFERENCES workflow_templates(id) ON DELETE SET NULL,
    workflow_stage        TEXT,
    task_type             TEXT,
    capability_id         UUID REFERENCES capabilities(id) ON DELETE SET NULL,
    risk_level            TEXT NOT NULL DEFAULT 'low'
        CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    overall_score         DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    expertise_score       DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    track_record_score    DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    reliability_score     DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    recency_score         DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    context_fit_score     DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    principle_score       DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    decision_count        INT NOT NULL DEFAULT 0,
    context               JSONB NOT NULL DEFAULT '{}',
    last_updated          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (actor_id, actor_type, scope_hash)
);

CREATE TABLE IF NOT EXISTS capability_evaluations (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    capability_id         UUID REFERENCES capabilities(id) ON DELETE SET NULL,
    actor_id              UUID,
    actor_type            TEXT,
    workflow_id           UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id               UUID REFERENCES tasks(id) ON DELETE SET NULL,
    evaluator_id          UUID,
    evaluator_type        TEXT NOT NULL DEFAULT 'human',
    quality_score         DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    reliability_score     DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    cost_score            DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    latency_score         DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    risk_score            DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    compliance_score      DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    overall_score         DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    evidence              JSONB NOT NULL DEFAULT '{}',
    conclusion            TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (capability_id IS NOT NULL OR actor_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_ai_agents_origin ON ai_agents(agent_origin, service_class, risk_level);
CREATE INDEX IF NOT EXISTS idx_employee_profiles_status ON employee_profiles(status, employment_type);
CREATE INDEX IF NOT EXISTS idx_access_decisions_actor ON access_decisions(actor_id, actor_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_decisions_resource ON access_decisions(resource, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_decisions_context ON access_decisions(organization_id, department_id, risk_level);
CREATE INDEX IF NOT EXISTS idx_context_weights_actor ON context_weight_scores(actor_id, actor_type, overall_score DESC);
CREATE INDEX IF NOT EXISTS idx_context_weights_scope ON context_weight_scores(scope_hash, risk_level);
CREATE INDEX IF NOT EXISTS idx_capability_evaluations_capability ON capability_evaluations(capability_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_capability_evaluations_actor ON capability_evaluations(actor_id, actor_type, created_at DESC);
