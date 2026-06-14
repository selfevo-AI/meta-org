-- 015_single_org_positions_workflow_graph.sql

CREATE TABLE IF NOT EXISTS positions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    department_id         UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    code                  TEXT,
    description           TEXT NOT NULL DEFAULT '',
    status                TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    sort_order            INT NOT NULL DEFAULT 0,
    permission_level      TEXT NOT NULL DEFAULT 'L1'
        CHECK (permission_level IN ('L1', 'L2', 'L3', 'L4')),
    required_capabilities JSONB NOT NULL DEFAULT '[]',
    metadata              JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_positions_department_code
    ON positions(department_id, code)
    WHERE code IS NOT NULL AND code <> '';

CREATE INDEX IF NOT EXISTS idx_positions_org_department ON positions(organization_id, department_id, status, sort_order);
CREATE INDEX IF NOT EXISTS idx_positions_permission ON positions(permission_level, status);

CREATE TABLE IF NOT EXISTS position_assignments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id        UUID NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    organization_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    department_id      UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    actor_id           UUID NOT NULL,
    actor_type         TEXT NOT NULL
        CHECK (actor_type IN ('internal_human', 'external_human', 'internal_agent', 'external_agent')),
    assignment_type    TEXT NOT NULL DEFAULT 'candidate'
        CHECK (assignment_type IN ('primary', 'backup', 'candidate')),
    allocation_percent NUMERIC(5,2) NOT NULL DEFAULT 100,
    status             TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    metadata           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_position_assignment_actor
    ON position_assignments(position_id, actor_id, actor_type)
    WHERE status <> 'archived';

CREATE INDEX IF NOT EXISTS idx_position_assignments_position ON position_assignments(position_id, status);
CREATE INDEX IF NOT EXISTS idx_position_assignments_actor ON position_assignments(actor_id, actor_type, status);

ALTER TABLE workflow_templates
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS visual_graph JSONB NOT NULL DEFAULT '{}';

ALTER TABLE workflow_instances
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;

ALTER TABLE project_members
    ADD COLUMN IF NOT EXISTS position_id UUID REFERENCES positions(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS position_assignment_id UUID REFERENCES position_assignments(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workflow_templates_org_department ON workflow_templates(organization_id, department_id, is_active);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_org_project ON workflow_instances(organization_id, project_id, status);
CREATE INDEX IF NOT EXISTS idx_project_members_position ON project_members(position_id, position_assignment_id);
