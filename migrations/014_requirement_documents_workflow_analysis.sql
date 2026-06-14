-- 014_requirement_documents_workflow_analysis.sql

CREATE TABLE IF NOT EXISTS requirement_documents (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_id    UUID NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
    file_name         TEXT NOT NULL,
    content_type      TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes        BIGINT NOT NULL DEFAULT 0,
    uploaded_by_id    UUID,
    uploaded_by_type  TEXT NOT NULL DEFAULT '',
    content           BYTEA NOT NULL,
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS requirement_analysis_workflows (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requirement_id        UUID NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
    workflow_id           UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    workflow_template_id  UUID NOT NULL REFERENCES workflow_templates(id) ON DELETE CASCADE,
    status               TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'paused', 'completed', 'failed', 'cancelled')),
    analysis_result      JSONB NOT NULL DEFAULT '{}',
    metadata             JSONB NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (requirement_id, workflow_id)
);

CREATE INDEX IF NOT EXISTS idx_requirement_documents_requirement ON requirement_documents(requirement_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_requirement_analysis_workflows_requirement ON requirement_analysis_workflows(requirement_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_requirement_analysis_workflows_workflow ON requirement_analysis_workflows(workflow_id);
