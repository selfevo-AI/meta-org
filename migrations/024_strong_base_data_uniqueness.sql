CREATE OR REPLACE FUNCTION assert_no_duplicate_key(check_name TEXT, duplicate_query TEXT)
RETURNS VOID AS $$
DECLARE
    duplicate_keys TEXT;
BEGIN
    EXECUTE duplicate_query INTO duplicate_keys;
    IF duplicate_keys IS NOT NULL AND duplicate_keys <> '' THEN
        RAISE EXCEPTION 'duplicate base data found for %: %', check_name, duplicate_keys;
    END IF;
END;
$$ LANGUAGE plpgsql;

SELECT assert_no_duplicate_key('organizations.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT lower(btrim(name)) AS key
        FROM organizations
        GROUP BY lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('muvrs.organization_id.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT organization_id::text || ':' || lower(btrim(name)) AS key
        FROM muvrs
        GROUP BY organization_id, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('teams.mvru_id.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT mvru_id::text || ':' || lower(btrim(name)) AS key
        FROM teams
        GROUP BY mvru_id, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('departments.organization_id.code', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT organization_id::text || ':' || lower(btrim(code)) AS key
        FROM departments
        WHERE code IS NOT NULL AND btrim(code) <> ''
        GROUP BY organization_id, lower(btrim(code))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('departments.organization_id.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT organization_id::text || ':' || lower(btrim(name)) AS key
        FROM departments
        GROUP BY organization_id, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('positions.department_id.code', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT department_id::text || ':' || lower(btrim(code)) AS key
        FROM positions
        WHERE code IS NOT NULL AND btrim(code) <> ''
        GROUP BY department_id, lower(btrim(code))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('positions.department_id.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT department_id::text || ':' || lower(btrim(name)) AS key
        FROM positions
        GROUP BY department_id, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('external_members.email', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT lower(btrim(email)) AS key
        FROM external_members
        WHERE email IS NOT NULL AND btrim(email) <> ''
        GROUP BY lower(btrim(email))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('capabilities.name.version', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT lower(btrim(name)) || ':' || version AS key
        FROM capabilities
        GROUP BY lower(btrim(name)), version
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('workflow_templates.organization.department.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT COALESCE(organization_id::text, '') || ':' || COALESCE(department_id::text, '') || ':' || lower(btrim(name)) AS key
        FROM workflow_templates
        GROUP BY organization_id, department_id, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('model_providers.provider_type.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT provider_type || ':' || lower(btrim(name)) AS key
        FROM model_providers
        GROUP BY provider_type, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('models.provider_id.model_key', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT provider_id::text || ':' || lower(btrim(model_key)) AS key
        FROM models
        GROUP BY provider_id, lower(btrim(model_key))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('model_provider_channels.provider_id.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT provider_id::text || ':' || lower(btrim(name)) AS key
        FROM model_provider_channels
        GROUP BY provider_id, lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('finance_adapters.name', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT lower(btrim(name)) AS key
        FROM finance_adapters
        GROUP BY lower(btrim(name))
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('cost_rate_cards.business_key', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT subject_type || ':' || COALESCE(subject_id::text, '') || ':' || scope_type || ':' || COALESCE(scope_id::text, '') || ':' || rate_type || ':' || currency || ':' || effective_from::text AS key
        FROM cost_rate_cards
        GROUP BY subject_type, subject_id, scope_type, scope_id, rate_type, currency, effective_from
        HAVING COUNT(*) > 1
    ) d
$$);

SELECT assert_no_duplicate_key('cost_budgets.business_key', $$
    SELECT string_agg(key, '; ') FROM (
        SELECT scope_type || ':' || COALESCE(scope_id::text, '') || ':' || currency || ':' || COALESCE(period_start::text, '') || ':' || COALESCE(period_end::text, '') AS key
        FROM cost_budgets
        GROUP BY scope_type, scope_id, currency, period_start, period_end
        HAVING COUNT(*) > 1
    ) d
$$);

CREATE UNIQUE INDEX IF NOT EXISTS uq_organizations_name_ci
    ON organizations (lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_muvrs_org_name_ci
    ON muvrs (organization_id, lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_teams_mvru_name_ci
    ON teams (mvru_id, lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_org_code_ci
    ON departments (organization_id, lower(btrim(code)))
    WHERE code IS NOT NULL AND btrim(code) <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_org_name_ci
    ON departments (organization_id, lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_positions_department_code_ci
    ON positions (department_id, lower(btrim(code)))
    WHERE code IS NOT NULL AND btrim(code) <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_positions_department_name_ci
    ON positions (department_id, lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_external_members_email_ci
    ON external_members (lower(btrim(email)))
    WHERE email IS NOT NULL AND btrim(email) <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_capabilities_name_version_ci
    ON capabilities (lower(btrim(name)), version);

CREATE UNIQUE INDEX IF NOT EXISTS uq_workflow_templates_scope_name_ci
    ON workflow_templates (COALESCE(organization_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(department_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_model_providers_type_name_ci
    ON model_providers (provider_type, lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_models_provider_key_ci
    ON models (provider_id, lower(btrim(model_key)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_model_provider_channels_provider_name_ci
    ON model_provider_channels (provider_id, lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_adapters_name_ci
    ON finance_adapters (lower(btrim(name)));

CREATE UNIQUE INDEX IF NOT EXISTS uq_cost_rate_cards_business_key
    ON cost_rate_cards (
        subject_type,
        COALESCE(subject_id, '00000000-0000-0000-0000-000000000000'::uuid),
        scope_type,
        COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid),
        rate_type,
        currency,
        effective_from
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_cost_budgets_business_key
    ON cost_budgets (
        scope_type,
        COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid),
        currency,
        COALESCE(period_start, '-infinity'::timestamptz),
        COALESCE(period_end, 'infinity'::timestamptz)
    );

DROP FUNCTION assert_no_duplicate_key(TEXT, TEXT);
