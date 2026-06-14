-- 011_organization_tree.sql

CREATE TABLE IF NOT EXISTS departments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    parent_id       UUID REFERENCES departments(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    code            TEXT,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    sort_order      INT NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_department_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_org_code
    ON departments(organization_id, code)
    WHERE code IS NOT NULL AND code <> '';

CREATE INDEX IF NOT EXISTS idx_departments_org_parent ON departments(organization_id, parent_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_departments_status ON departments(status);

CREATE TABLE IF NOT EXISTS external_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    email           TEXT,
    vendor          TEXT NOT NULL DEFAULT '',
    contract_type   TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_external_members_email
    ON external_members(email)
    WHERE email IS NOT NULL AND email <> '';

CREATE INDEX IF NOT EXISTS idx_external_members_status ON external_members(status);

CREATE TABLE IF NOT EXISTS organization_memberships (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    department_id       UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    member_type         TEXT NOT NULL CHECK (member_type IN ('internal', 'external', 'agent')),
    user_id             UUID REFERENCES users(id) ON DELETE CASCADE,
    external_member_id  UUID REFERENCES external_members(id) ON DELETE CASCADE,
    agent_id            UUID REFERENCES ai_agents(id) ON DELETE CASCADE,
    title               TEXT NOT NULL DEFAULT '',
    role_id             UUID REFERENCES roles(id) ON DELETE SET NULL,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_organization_membership_actor CHECK (
        (member_type = 'internal' AND user_id IS NOT NULL AND external_member_id IS NULL AND agent_id IS NULL) OR
        (member_type = 'external' AND user_id IS NULL AND external_member_id IS NOT NULL AND agent_id IS NULL) OR
        (member_type = 'agent' AND user_id IS NULL AND external_member_id IS NULL AND agent_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_org_membership_internal
    ON organization_memberships(department_id, user_id)
    WHERE member_type = 'internal';

CREATE UNIQUE INDEX IF NOT EXISTS uq_org_membership_external
    ON organization_memberships(department_id, external_member_id)
    WHERE member_type = 'external';

CREATE UNIQUE INDEX IF NOT EXISTS uq_org_membership_agent
    ON organization_memberships(department_id, agent_id)
    WHERE member_type = 'agent';

CREATE INDEX IF NOT EXISTS idx_org_memberships_org ON organization_memberships(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_memberships_department ON organization_memberships(department_id);
CREATE INDEX IF NOT EXISTS idx_org_memberships_type_status ON organization_memberships(member_type, status);

CREATE TABLE IF NOT EXISTS department_mvru_links (
    department_id   UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    mvru_id         UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    link_type       TEXT NOT NULL DEFAULT 'execution',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (department_id, mvru_id, link_type)
);

CREATE INDEX IF NOT EXISTS idx_department_mvru_links_mvru ON department_mvru_links(mvru_id);
