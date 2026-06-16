ALTER TABLE position_assignments
    ADD COLUMN IF NOT EXISTS meta_resource_id UUID REFERENCES meta_resources(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_position_assignments_meta_resource
    ON position_assignments(meta_resource_id, status);

CREATE TABLE IF NOT EXISTS task_matrix_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    position_id UUID NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    position_assignment_id UUID REFERENCES position_assignments(id) ON DELETE SET NULL,
    meta_resource_id UUID NOT NULL REFERENCES meta_resources(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL,
    actor_type TEXT NOT NULL
        CHECK (actor_type IN ('internal_human', 'external_human', 'internal_agent', 'external_agent')),
    role_in_task TEXT NOT NULL DEFAULT 'owner'
        CHECK (role_in_task IN ('owner', 'reviewer', 'support', 'observer')),
    allocation_percent NUMERIC(5,2) NOT NULL DEFAULT 100,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_task_matrix_assignment_role
    ON task_matrix_assignments(task_id, position_id, meta_resource_id, role_in_task)
    WHERE status <> 'archived';

CREATE INDEX IF NOT EXISTS idx_task_matrix_assignments_task
    ON task_matrix_assignments(task_id, role_in_task, status);

CREATE INDEX IF NOT EXISTS idx_task_matrix_assignments_meta_resource
    ON task_matrix_assignments(meta_resource_id, status);

CREATE INDEX IF NOT EXISTS idx_task_matrix_assignments_position
    ON task_matrix_assignments(position_id, position_assignment_id, status);
