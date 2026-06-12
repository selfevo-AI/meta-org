-- 005_capability.sql

CREATE TABLE capabilities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    version         VARCHAR(50) NOT NULL DEFAULT '1.0',
    description     TEXT,
    input_schema    JSONB NOT NULL DEFAULT '{}',
    output_schema   JSONB NOT NULL DEFAULT '{}',
    preconditions   JSONB NOT NULL DEFAULT '[]',
    error_handling  JSONB NOT NULL DEFAULT '{}',
    permission_level permission_level NOT NULL DEFAULT 'L2',
    cost_estimate   JSONB NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (name, version)
);

CREATE TABLE capability_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    capability_id   UUID NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    mvru_id         UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    config          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (capability_id, mvru_id)
);

CREATE TABLE capability_invocations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    capability_id   UUID NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    caller_id       UUID NOT NULL,
    caller_type     VARCHAR(10) NOT NULL,
    input           JSONB,
    output          JSONB,
    duration_ms     INT,
    cost            NUMERIC(12,4),
    outcome         VARCHAR(20),
    trace_id        UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cap_name ON capabilities(name);
CREATE INDEX idx_cap_bind_mvru ON capability_bindings(mvru_id);
CREATE INDEX idx_cap_inv_caller ON capability_invocations(caller_id);
