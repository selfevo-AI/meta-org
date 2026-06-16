CREATE TABLE IF NOT EXISTS assistant_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL DEFAULT '',
    mode TEXT NOT NULL DEFAULT 'business_process'
        CHECK (mode IN ('business_process', 'self_evolution')),
    module_key TEXT NOT NULL DEFAULT 'general',
    status TEXT NOT NULL DEFAULT 'idle'
        CHECK (status IN ('idle', 'running', 'approval_required', 'completed', 'failed', 'cancelled')),
    actor_id UUID NOT NULL,
    actor_type TEXT NOT NULL,
    provider_id UUID REFERENCES model_providers(id) ON DELETE SET NULL,
    preferred_channel_id UUID REFERENCES model_provider_channels(id) ON DELETE SET NULL,
    provider_type TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    service_tier TEXT NOT NULL DEFAULT '',
    reasoning_effort TEXT NOT NULL DEFAULT '',
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    position_id UUID REFERENCES positions(id) ON DELETE SET NULL,
    position_assignment_id UUID REFERENCES position_assignments(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    working_memory JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assistant_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES assistant_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT NOT NULL DEFAULT '',
    tool_call_id TEXT NOT NULL DEFAULT '',
    tool_name TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assistant_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES assistant_sessions(id) ON DELETE CASCADE,
    module_key TEXT NOT NULL DEFAULT 'general',
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    position_id UUID REFERENCES positions(id) ON DELETE SET NULL,
    position_assignment_id UUID REFERENCES position_assignments(id) ON DELETE SET NULL,
    invocation_id UUID REFERENCES ai_invocations(id) ON DELETE SET NULL,
    tool_execution_id UUID REFERENCES tool_executions(id) ON DELETE SET NULL,
    tool_approval_id UUID REFERENCES tool_approvals(id) ON DELETE SET NULL,
    step_type TEXT NOT NULL CHECK (step_type IN ('llm', 'tool_call', 'tool_result', 'memory', 'approval', 'error')),
    status TEXT NOT NULL DEFAULT 'completed',
    summary TEXT NOT NULL DEFAULT '',
    data JSONB NOT NULL DEFAULT '{}',
    turn INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assistant_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_key TEXT NOT NULL DEFAULT 'general',
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    position_id UUID REFERENCES positions(id) ON DELETE SET NULL,
    position_assignment_id UUID REFERENCES position_assignments(id) ON DELETE SET NULL,
    actor_id UUID,
    actor_type TEXT NOT NULL DEFAULT '',
    memory_type TEXT NOT NULL DEFAULT 'lesson'
        CHECK (memory_type IN ('working', 'knowledge', 'preference', 'lesson')),
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    data JSONB NOT NULL DEFAULT '{}',
    source_session_id UUID REFERENCES assistant_sessions(id) ON DELETE SET NULL,
    source_step_id UUID REFERENCES assistant_steps(id) ON DELETE SET NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assistant_sessions_actor
    ON assistant_sessions(actor_id, actor_type, module_key, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_sessions_scope
    ON assistant_sessions(module_key, organization_id, department_id, position_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_sessions_status
    ON assistant_sessions(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_messages_session
    ON assistant_messages(session_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_assistant_steps_session
    ON assistant_steps(session_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_assistant_steps_scope
    ON assistant_steps(module_key, organization_id, department_id, position_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_memories_scope
    ON assistant_memories(module_key, organization_id, department_id, position_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_memories_actor
    ON assistant_memories(actor_id, actor_type, module_key, updated_at DESC);

INSERT INTO tool_definitions (name, description, source_type, default_policy, risk_level, required_level, input_schema)
VALUES
    (
        'evolution.create_knowledge',
        'Create an evolution knowledge entry from verified assistant work',
        'internal_api',
        'notify',
        'medium',
        'L2',
        '{"type":"object","properties":{"title":{"type":"string"},"content":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}},"workflow_id":{"type":"string"}},"required":["title","content"]}'::jsonb
    ),
    (
        'evolution.create_signal',
        'Create an evolution signal for follow-up review',
        'internal_api',
        'notify',
        'medium',
        'L2',
        '{"type":"object","properties":{"signal_type":{"type":"string"},"source":{"type":"string"},"priority":{"type":"integer"},"data":{"type":"object"}},"required":["signal_type"]}'::jsonb
    ),
    (
        'evolution.propose_experiment',
        'Propose a system evolution experiment',
        'internal_api',
        'approve',
        'high',
        'L3',
        '{"type":"object","properties":{"name":{"type":"string"},"hypothesis":{"type":"string"},"success_criteria":{"type":"object"}},"required":["name","hypothesis"]}'::jsonb
    )
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    source_type = EXCLUDED.source_type,
    default_policy = EXCLUDED.default_policy,
    risk_level = EXCLUDED.risk_level,
    required_level = EXCLUDED.required_level,
    input_schema = EXCLUDED.input_schema,
    updated_at = NOW();
