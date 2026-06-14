-- 003_organization.sql

CREATE TABLE IF NOT EXISTS organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$ BEGIN
    CREATE TYPE mvru_status AS ENUM ('designing', 'active', 'evaluating', 'evolving', 'dissolved');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS muvrs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    status          mvru_status NOT NULL DEFAULT 'designing',
    boundary        JSONB NOT NULL DEFAULT '{"data_permissions":[],"resource_quota":{},"network_policies":[]}',
    config          JSONB NOT NULL DEFAULT '{}',
    parent_id       UUID REFERENCES muvrs(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mvru_id     UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mvru_members (
    mvru_id     UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    agent_id    UUID REFERENCES ai_agents(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT chk_one_actor CHECK (
        (user_id IS NOT NULL AND agent_id IS NULL) OR
        (user_id IS NULL AND agent_id IS NOT NULL)
    ),
    CONSTRAINT uq_mvru_user UNIQUE (mvru_id, user_id),
    CONSTRAINT uq_mvru_agent UNIQUE (mvru_id, agent_id)
);

CREATE TABLE IF NOT EXISTS mvru_relationships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_mvru_id  UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    target_mvru_id  UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    rel_type        VARCHAR(50) NOT NULL DEFAULT 'collaborate',
    config          JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_no_self_ref CHECK (source_mvru_id != target_mvru_id)
);

CREATE INDEX IF NOT EXISTS idx_muvrs_org ON muvrs(organization_id);
CREATE INDEX IF NOT EXISTS idx_teams_mvru ON teams(mvru_id);
