CREATE TABLE IF NOT EXISTS tool_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL CHECK (source_type IN ('internal_api', 'interface_file', 'manual_approval')),
    default_policy TEXT NOT NULL DEFAULT 'approve' CHECK (default_policy IN ('auto', 'notify', 'approve', 'deny')),
    risk_level TEXT NOT NULL DEFAULT 'medium' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    required_level TEXT NOT NULL DEFAULT 'L1',
    input_schema JSONB NOT NULL DEFAULT '{}',
    output_schema JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS interface_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    file_type TEXT NOT NULL CHECK (file_type IN ('json', 'yaml', 'markdown')),
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tool_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id UUID NOT NULL REFERENCES tool_definitions(id) ON DELETE RESTRICT,
    invocation_id UUID REFERENCES ai_invocations(id) ON DELETE SET NULL,
    actor_id UUID NOT NULL,
    actor_type TEXT NOT NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    idempotency_key TEXT NOT NULL DEFAULT '',
    policy TEXT NOT NULL,
    governance_decision TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'approval_required', 'approved', 'running', 'completed', 'rejected', 'denied', 'failed')),
    arguments JSONB NOT NULL DEFAULT '{}',
    result_summary TEXT NOT NULL DEFAULT '',
    result JSONB NOT NULL DEFAULT '{}',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tool_execution_idempotency
    ON tool_executions(tool_id, idempotency_key)
    WHERE idempotency_key <> '';

CREATE TABLE IF NOT EXISTS tool_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES tool_executions(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'expired')),
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tool_definitions_active ON tool_definitions(is_active, source_type, name);
CREATE INDEX IF NOT EXISTS idx_tool_executions_status ON tool_executions(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tool_approvals_status ON tool_approvals(status, created_at DESC);

INSERT INTO tool_definitions (name, description, source_type, default_policy, risk_level, required_level)
VALUES
    ('requirement.analyze', 'Analyze a requirement', 'internal_api', 'notify', 'medium', 'L2'),
    ('project.match_members', 'Recommend project members', 'internal_api', 'notify', 'medium', 'L2'),
    ('project.bind_workflow', 'Bind workflow to project', 'internal_api', 'approve', 'high', 'L3'),
    ('project.estimate_cost', 'Estimate project cost', 'internal_api', 'notify', 'medium', 'L2'),
    ('project.create_cost_entry', 'Create project cost entry', 'internal_api', 'approve', 'high', 'L3'),
    ('governance.explain_decision', 'Explain governance decision', 'internal_api', 'notify', 'low', 'L1'),
    ('finance.prepare_export_batch', 'Prepare finance export batch', 'manual_approval', 'approve', 'high', 'L3')
ON CONFLICT (name) DO NOTHING;
