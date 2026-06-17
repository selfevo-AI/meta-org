ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE organization_memberships
    ADD COLUMN IF NOT EXISTS authority_tier TEXT NOT NULL DEFAULT 'executor'
        CHECK (authority_tier IN ('organization_creator', 'reviewer', 'executor'));

ALTER TABLE tool_definitions
    ADD COLUMN IF NOT EXISTS tool_category TEXT NOT NULL DEFAULT 'execution_operation'
        CHECK (tool_category IN ('core_data', 'business_approval', 'execution_operation')),
    ADD COLUMN IF NOT EXISTS approval_tier_required TEXT NOT NULL DEFAULT 'executor'
        CHECK (approval_tier_required IN ('organization_creator', 'reviewer', 'executor'));

ALTER TABLE tool_executions
    ADD COLUMN IF NOT EXISTS requested_by_human_id UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE tool_approvals
    ADD COLUMN IF NOT EXISTS approved_by_human_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_organizations_created_by
    ON organizations(created_by)
    WHERE created_by IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_org_memberships_authority
    ON organization_memberships(organization_id, user_id, authority_tier, status)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tool_definitions_authority
    ON tool_definitions(tool_category, approval_tier_required, required_level);

UPDATE tool_definitions
SET tool_category = 'core_data',
    approval_tier_required = 'organization_creator',
    required_level = CASE
        WHEN required_level IN ('', 'L1', 'L2', 'L3') THEN 'L4'
        ELSE required_level
    END,
    default_policy = 'approve',
    updated_at = NOW()
WHERE name LIKE 'model.%'
   OR name LIKE 'organization.%'
   OR name LIKE 'department.%'
   OR name LIKE 'position.%'
   OR name LIKE 'governance.%'
   OR name LIKE 'tool.%'
   OR name LIKE 'costing.%';

UPDATE tool_definitions
SET tool_category = 'business_approval',
    approval_tier_required = 'reviewer',
    required_level = CASE
        WHEN required_level IN ('', 'L1', 'L2') THEN 'L3'
        ELSE required_level
    END,
    default_policy = 'approve',
    updated_at = NOW()
WHERE name LIKE '%.approve%'
   OR name LIKE 'workflow.%'
   OR name LIKE 'stage.%'
   OR name LIKE 'finance.%'
   OR name IN ('project.bind_workflow', 'project.create_cost_entry', 'evolution.propose_experiment');

UPDATE tool_definitions
SET tool_category = 'execution_operation',
    approval_tier_required = 'executor',
    updated_at = NOW()
WHERE tool_category NOT IN ('core_data', 'business_approval');
