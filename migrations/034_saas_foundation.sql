-- 034_saas_foundation.sql
-- SaaS mode foundation: platform administration, onboarding, invitations,
-- module entitlements, subscription metadata, and organization-scoped roles.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS account_status TEXT NOT NULL DEFAULT 'active'
        CHECK (account_status IN ('active', 'disabled')),
    ADD COLUMN IF NOT EXISTS onboarding_status TEXT NOT NULL DEFAULT 'required'
        CHECK (onboarding_status IN ('required', 'complete')),
    ADD COLUMN IF NOT EXISTS default_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

UPDATE users u
SET onboarding_status = 'complete',
    default_organization_id = COALESCE(u.default_organization_id, first_org.organization_id)
FROM (
    SELECT DISTINCT ON (user_id) user_id, organization_id
    FROM organization_memberships
    WHERE user_id IS NOT NULL AND status = 'active'
    ORDER BY user_id, joined_at ASC
) first_org
WHERE u.id = first_org.user_id
  AND u.onboarding_status = 'required';

CREATE INDEX IF NOT EXISTS idx_users_onboarding
    ON users(onboarding_status, default_organization_id);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        GROUP BY lower(email)
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate user emails found when compared case-insensitively; resolve duplicates before applying SaaS foundation migration';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_lower
    ON users(lower(email));

CREATE TABLE IF NOT EXISTS platform_admins (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'system_owner'
        CHECK (role IN ('system_owner', 'system_admin', 'support')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saas_modules (
    module_key      TEXT PRIMARY KEY,
    display_name    TEXT NOT NULL,
    category        TEXT NOT NULL DEFAULT 'business',
    enabled_default BOOLEAN NOT NULL DEFAULT true,
    license_scope   TEXT NOT NULL DEFAULT 'mit'
        CHECK (license_scope IN ('mit', 'commercial')),
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saas_plans (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code           TEXT NOT NULL UNIQUE,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'archived')),
    price_amount   NUMERIC(14,2) NOT NULL DEFAULT 0,
    currency       TEXT NOT NULL DEFAULT 'CNY',
    billing_cycle  TEXT NOT NULL DEFAULT 'monthly'
        CHECK (billing_cycle IN ('monthly', 'yearly', 'manual')),
    metadata       JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saas_plan_modules (
    plan_id    UUID NOT NULL REFERENCES saas_plans(id) ON DELETE CASCADE,
    module_key TEXT NOT NULL REFERENCES saas_modules(module_key) ON DELETE CASCADE,
    limit_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plan_id, module_key)
);

CREATE TABLE IF NOT EXISTS organization_subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id         UUID REFERENCES saas_plans(id) ON DELETE SET NULL,
    status          TEXT NOT NULL DEFAULT 'trialing'
        CHECK (status IN ('trialing', 'active', 'past_due', 'cancelled', 'expired')),
    trial_ends_at   TIMESTAMPTZ,
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS organization_module_entitlements (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    module_key      TEXT NOT NULL REFERENCES saas_modules(module_key) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'enabled'
        CHECK (status IN ('enabled', 'disabled')),
    source          TEXT NOT NULL DEFAULT 'plan'
        CHECK (source IN ('plan', 'manual', 'trial')),
    limit_json      JSONB NOT NULL DEFAULT '{}',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, module_key)
);

CREATE INDEX IF NOT EXISTS idx_org_entitlements_status
    ON organization_module_entitlements(module_key, status);

CREATE TABLE IF NOT EXISTS organization_usage_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    module_key      TEXT NOT NULL REFERENCES saas_modules(module_key) ON DELETE CASCADE,
    usage_key       TEXT NOT NULL,
    quantity        NUMERIC(18,6) NOT NULL DEFAULT 1,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata        JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_org_usage_events_lookup
    ON organization_usage_events(organization_id, module_key, usage_key, occurred_at DESC);

CREATE TABLE IF NOT EXISTS organization_invitations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email           TEXT NOT NULL,
    name            TEXT NOT NULL DEFAULT '',
    role_id         UUID REFERENCES roles(id) ON DELETE SET NULL,
    authority_tier  TEXT NOT NULL DEFAULT 'executor'
        CHECK (authority_tier IN ('organization_creator', 'organization_admin', 'reviewer', 'executor')),
    token_hash      TEXT NOT NULL UNIQUE,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    invited_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    accepted_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    accepted_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_invitations_org_status
    ON organization_invitations(organization_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_org_invitations_email_status
    ON organization_invitations(lower(email), status);

DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'organization_memberships'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%authority_tier%';

    IF constraint_name IS NOT NULL THEN
        EXECUTE FORMAT('ALTER TABLE organization_memberships DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

ALTER TABLE organization_memberships
    ADD CONSTRAINT chk_organization_memberships_authority_tier
    CHECK (authority_tier IN ('organization_creator', 'organization_admin', 'reviewer', 'executor'));

INSERT INTO saas_modules(module_key, display_name, category, enabled_default, license_scope) VALUES
    ('organization', 'Organization', 'base_data', true, 'mit'),
    ('project', 'Project Operations', 'business', true, 'mit'),
    ('workflow', 'Workflow', 'business', true, 'mit'),
    ('governance', 'Governance', 'governance', true, 'mit'),
    ('evolution', 'Evolution', 'governance', true, 'mit'),
    ('capability', 'Capability', 'base_data', true, 'mit'),
    ('meta_resource', 'Meta Resource', 'base_data', true, 'mit'),
    ('assistant', 'AI Assistant', 'ai', true, 'commercial'),
    ('ai_gateway', 'AI Gateway', 'ai', true, 'commercial'),
    ('toolruntime', 'Tool Runtime', 'ai', true, 'commercial'),
    ('finance', 'Finance', 'finance', true, 'commercial'),
    ('costing', 'Costing', 'finance', true, 'mit'),
    ('developer_tools', 'Developer Tools', 'system', true, 'commercial')
ON CONFLICT (module_key) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    category = EXCLUDED.category,
    enabled_default = EXCLUDED.enabled_default,
    license_scope = EXCLUDED.license_scope,
    updated_at = NOW();

INSERT INTO saas_plans(code, name, description, status, price_amount, currency, billing_cycle, metadata)
VALUES ('foundation', 'Foundation', 'Default SaaS foundation plan with all current modules enabled.', 'active', 0, 'CNY', 'manual', '{"default":true}'::jsonb)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();

INSERT INTO saas_plan_modules(plan_id, module_key)
SELECT p.id, m.module_key
FROM saas_plans p
CROSS JOIN saas_modules m
WHERE p.code = 'foundation'
ON CONFLICT (plan_id, module_key) DO NOTHING;
