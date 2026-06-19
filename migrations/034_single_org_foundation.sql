-- 034_single_org_identity_foundation.sql
-- Single-organization distribution foundation: account status, default organization,
-- case-insensitive user email uniqueness, and organization authority tiers.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS account_status TEXT NOT NULL DEFAULT 'active'
        CHECK (account_status IN ('active', 'disabled')),
    ADD COLUMN IF NOT EXISTS onboarding_status TEXT NOT NULL DEFAULT 'complete'
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
WHERE u.id = first_org.user_id;

UPDATE users
SET onboarding_status = 'complete'
WHERE onboarding_status = 'required'
  AND default_organization_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_default_organization
    ON users(default_organization_id);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        GROUP BY lower(email)
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate user emails found when compared case-insensitively; resolve duplicates before applying single-org identity migration';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_lower
    ON users(lower(email));

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
