-- 013_project_lifecycle.sql

CREATE TABLE IF NOT EXISTS requirements (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    source           TEXT NOT NULL DEFAULT 'manual',
    status           TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'analyzed', 'approved', 'converted', 'rejected', 'archived')),
    priority         TEXT NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    risk_level       TEXT NOT NULL DEFAULT 'low'
        CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    required_level   TEXT NOT NULL DEFAULT 'L1'
        CHECK (required_level IN ('L1', 'L2', 'L3', 'L4')),
    organization_id  UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id    UUID REFERENCES departments(id) ON DELETE SET NULL,
    created_by_id    UUID,
    created_by_type  TEXT NOT NULL DEFAULT '',
    analysis         JSONB NOT NULL DEFAULT '{}',
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS projects (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_id   UUID REFERENCES requirements(id) ON DELETE SET NULL,
    organization_id  UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id    UUID REFERENCES departments(id) ON DELETE SET NULL,
    name             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'planning'
        CHECK (status IN ('planning', 'active', 'paused', 'delivering', 'completed', 'closed', 'cancelled')),
    priority         TEXT NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    risk_level       TEXT NOT NULL DEFAULT 'low'
        CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    required_level   TEXT NOT NULL DEFAULT 'L1'
        CHECK (required_level IN ('L1', 'L2', 'L3', 'L4')),
    budget_amount    NUMERIC(14,2) NOT NULL DEFAULT 0,
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_members (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id         UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id           UUID NOT NULL,
    actor_type         TEXT NOT NULL
        CHECK (actor_type IN ('internal_human', 'external_human', 'internal_agent', 'external_agent')),
    role               TEXT NOT NULL DEFAULT 'contributor',
    title              TEXT NOT NULL DEFAULT '',
    allocation_percent NUMERIC(5,2) NOT NULL DEFAULT 100,
    cost_rate          NUMERIC(14,2) NOT NULL DEFAULT 0,
    permission_level   TEXT NOT NULL DEFAULT 'L1'
        CHECK (permission_level IN ('L1', 'L2', 'L3', 'L4')),
    capabilities       JSONB NOT NULL DEFAULT '[]',
    status             TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    metadata           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_workflows (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id           UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_id          UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    workflow_template_id UUID REFERENCES workflow_templates(id) ON DELETE SET NULL,
    purpose              TEXT NOT NULL DEFAULT 'delivery',
    status               TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'paused', 'completed', 'cancelled')),
    metadata             JSONB NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deliverables (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    deliverable_type    TEXT NOT NULL DEFAULT 'artifact',
    uri                 TEXT NOT NULL DEFAULT '',
    version             TEXT NOT NULL DEFAULT '1.0',
    status              TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'submitted', 'accepted', 'rejected', 'archived')),
    submitted_by_id     UUID,
    submitted_by_type   TEXT NOT NULL DEFAULT '',
    accepted_by_id      UUID,
    accepted_by_type    TEXT NOT NULL DEFAULT '',
    evidence            JSONB NOT NULL DEFAULT '{}',
    metadata            JSONB NOT NULL DEFAULT '{}',
    submitted_at        TIMESTAMPTZ,
    accepted_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_cost_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_type  TEXT NOT NULL DEFAULT 'manual',
    source_id    UUID,
    actor_id     UUID,
    actor_type   TEXT NOT NULL DEFAULT '',
    amount       NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency     TEXT NOT NULL DEFAULT 'CNY',
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    description  TEXT NOT NULL DEFAULT '',
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_evaluations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id            UUID,
    actor_type          TEXT NOT NULL DEFAULT '',
    capability_id       UUID REFERENCES capabilities(id) ON DELETE SET NULL,
    evaluator_id        UUID,
    evaluator_type      TEXT NOT NULL DEFAULT 'internal_human',
    quality_score       DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    delivery_score      DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    cost_score          DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    collaboration_score DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    overall_score       DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    conclusion          TEXT NOT NULL DEFAULT '',
    evidence            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_requirements_status ON requirements(status, priority, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_requirements_org_department ON requirements(organization_id, department_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status, priority, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_requirement ON projects(requirement_id);
CREATE INDEX IF NOT EXISTS idx_projects_org_department ON projects(organization_id, department_id);
CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id, status);
CREATE INDEX IF NOT EXISTS idx_project_members_actor ON project_members(actor_id, actor_type);
CREATE INDEX IF NOT EXISTS idx_project_workflows_project ON project_workflows(project_id, status);
CREATE INDEX IF NOT EXISTS idx_deliverables_project ON deliverables(project_id, status);
CREATE INDEX IF NOT EXISTS idx_project_cost_entries_project ON project_cost_entries(project_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_evaluations_project ON project_evaluations(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_evaluations_actor ON project_evaluations(actor_id, actor_type, created_at DESC);
