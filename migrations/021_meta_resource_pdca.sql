CREATE TABLE IF NOT EXISTS meta_resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL
        CHECK (resource_type IN ('human', 'external_human', 'agent', 'model_channel', 'tool', 'material', 'time', 'capability', 'budget', 'resource')),
    source_type TEXT NOT NULL DEFAULT '',
    source_id UUID,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'reserved', 'exhausted', 'archived')),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    owner_actor_id UUID,
    owner_actor_type TEXT NOT NULL DEFAULT '',
    capability_profile JSONB NOT NULL DEFAULT '{}',
    cost_profile JSONB NOT NULL DEFAULT '{}',
    capacity_profile JSONB NOT NULL DEFAULT '{}',
    risk_profile JSONB NOT NULL DEFAULT '{}',
    performance_profile JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_meta_resources_unique_source
    ON meta_resources(resource_type, source_type, source_id)
    WHERE source_type <> '' AND source_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_meta_resources_type_status
    ON meta_resources(resource_type, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meta_resources_org_department
    ON meta_resources(organization_id, department_id);

CREATE TABLE IF NOT EXISTS demand_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_id UUID REFERENCES requirements(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    goal TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'planned', 'active', 'accepted', 'closed', 'archived')),
    acceptance_criteria JSONB NOT NULL DEFAULT '[]',
    required_capabilities JSONB NOT NULL DEFAULT '[]',
    budget_constraints JSONB NOT NULL DEFAULT '{}',
    time_constraints JSONB NOT NULL DEFAULT '{}',
    risk_constraints JSONB NOT NULL DEFAULT '{}',
    resource_fit_candidates JSONB NOT NULL DEFAULT '[]',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_demand_profiles_requirement
    ON demand_profiles(requirement_id);

CREATE INDEX IF NOT EXISTS idx_demand_profiles_project
    ON demand_profiles(project_id);

CREATE INDEX IF NOT EXISTS idx_demand_profiles_status
    ON demand_profiles(status, created_at DESC);

CREATE TABLE IF NOT EXISTS pdca_cycles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    demand_profile_id UUID REFERENCES demand_profiles(id) ON DELETE SET NULL,
    requirement_id UUID REFERENCES requirements(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'completed', 'cancelled', 'archived')),
    current_stage TEXT NOT NULL DEFAULT 'plan'
        CHECK (current_stage IN ('plan', 'do', 'change', 'accept')),
    outcome_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pdca_cycles_demand
    ON pdca_cycles(demand_profile_id);

CREATE INDEX IF NOT EXISTS idx_pdca_cycles_status_stage
    ON pdca_cycles(status, current_stage, created_at DESC);

CREATE TABLE IF NOT EXISTS pdca_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id UUID NOT NULL REFERENCES pdca_cycles(id) ON DELETE CASCADE,
    stage TEXT NOT NULL CHECK (stage IN ('plan', 'do', 'change', 'accept')),
    event_type TEXT NOT NULL DEFAULT 'note',
    source_type TEXT NOT NULL DEFAULT '',
    source_id UUID,
    actor_id UUID,
    actor_type TEXT NOT NULL DEFAULT '',
    resource_refs JSONB NOT NULL DEFAULT '[]',
    cost_refs JSONB NOT NULL DEFAULT '[]',
    evidence JSONB NOT NULL DEFAULT '{}',
    decision TEXT NOT NULL DEFAULT '',
    next_action TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pdca_events_cycle_stage
    ON pdca_events(cycle_id, stage, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pdca_events_source
    ON pdca_events(source_type, source_id);
