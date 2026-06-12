-- 006_workflow.sql

CREATE TYPE workflow_status AS ENUM ('active', 'paused', 'completed', 'failed');
CREATE TYPE task_status AS ENUM ('pending', 'assigned', 'in_progress', 'completed', 'rejected');
CREATE TYPE stage_type AS ENUM ('plan', 'execute', 'review');

CREATE TABLE workflow_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    stages          JSONB NOT NULL DEFAULT '[]',
    assignee_type   VARCHAR(10) NOT NULL DEFAULT 'either',
    required_weight NUMERIC(5,2) DEFAULT 0,
    routing_rules   JSONB NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id     UUID NOT NULL REFERENCES workflow_templates(id),
    status          workflow_status NOT NULL DEFAULT 'active',
    current_stage   INT NOT NULL DEFAULT 0,
    context         JSONB NOT NULL DEFAULT '{}',
    trace_id        UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id     UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    stage           INT NOT NULL,
    stage_type      stage_type NOT NULL,
    assignee_id     UUID,
    assignee_type   VARCHAR(10),
    input           JSONB,
    output          JSONB,
    weight_snapshot NUMERIC(5,2),
    status          task_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE decisions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id           UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    decision_maker_id UUID NOT NULL,
    maker_type        VARCHAR(10) NOT NULL,
    weight            NUMERIC(5,2) DEFAULT 0,
    input             JSONB,
    output            JSONB,
    reasoning         TEXT,
    outcome           VARCHAR(50),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_contexts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id       UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    working_memory    JSONB NOT NULL DEFAULT '{}',
    injected_experience JSONB NOT NULL DEFAULT '[]',
    principle_notes   TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workflow_id)
);

CREATE INDEX idx_tasks_workflow ON tasks(workflow_id);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_decisions_task ON decisions(task_id);
