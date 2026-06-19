-- 036_unified_skill.sql
-- Unified skill table: first-class skill governance while preserving existing assistant skill APIs.

CREATE TABLE IF NOT EXISTS skill (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_key TEXT NOT NULL,
    scope_level TEXT NOT NULL DEFAULT 'saas_global'
        CHECK (scope_level IN ('saas_global', 'organization', 'deployment')),
    deployment_mode TEXT NOT NULL DEFAULT 'saas'
        CHECK (deployment_mode IN ('saas', 'org_private', 'private')),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    module_key TEXT NOT NULL DEFAULT 'general',
    target_type TEXT NOT NULL DEFAULT '',
    business_function_key TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    trigger_intent TEXT NOT NULL DEFAULT '',
    prompt_template TEXT NOT NULL DEFAULT '',
    tool_allowlist JSONB NOT NULL DEFAULT '[]',
    input_schema JSONB NOT NULL DEFAULT '{}',
    output_schema JSONB NOT NULL DEFAULT '{}',
    skill_components JSONB NOT NULL DEFAULT '[]',
    permission_policy JSONB NOT NULL DEFAULT '{}',
    context_policy JSONB NOT NULL DEFAULT '{}',
    pricing_policy JSONB NOT NULL DEFAULT '{}',
    activation_policy JSONB NOT NULL DEFAULT '{}',
    version INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'archived')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by_type TEXT NOT NULL DEFAULT '',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    source_session_id UUID REFERENCES assistant_sessions(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (jsonb_typeof(skill_components) = 'array'),
    CHECK (jsonb_array_length(skill_components) BETWEEN 3 AND 9)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_skill_scope_key_version
    ON skill(scope_level, COALESCE(organization_id, '00000000-0000-0000-0000-000000000000'::uuid), skill_key, version);
CREATE INDEX IF NOT EXISTS idx_skill_scope_lookup
    ON skill(scope_level, organization_id, module_key, target_type, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_skill_business_function
    ON skill(module_key, business_function_key, status);

INSERT INTO skill (
    id, skill_key, scope_level, deployment_mode, organization_id, owner_user_id, module_key, target_type,
    business_function_key, name, description, trigger_intent, prompt_template, tool_allowlist,
    input_schema, output_schema, skill_components, permission_policy, context_policy, pricing_policy,
    activation_policy, version, status, created_by, created_by_type, reviewed_by, source_session_id,
    metadata, created_at, updated_at
)
SELECT
    abs.id,
    lower(regexp_replace(NULLIF(abs.name, ''), '[^a-zA-Z0-9]+', '_', 'g')) || '_' || left(abs.id::text, 8),
    'saas_global',
    'saas',
    NULL,
    abs.created_by,
    COALESCE(NULLIF(abs.module_key, ''), 'general'),
    COALESCE(abs.target_type, ''),
    COALESCE(NULLIF(abs.target_type, ''), COALESCE(NULLIF(abs.module_key, ''), 'general')),
    abs.name,
    COALESCE(abs.description, ''),
    COALESCE(abs.trigger_intent, ''),
    COALESCE(abs.prompt_template, ''),
    COALESCE(abs.tool_allowlist, '[]'::jsonb),
    COALESCE(abs.input_schema, '{}'::jsonb),
    COALESCE(abs.output_schema, '{}'::jsonb),
    jsonb_build_array(
        jsonb_build_object('key', 'intent', 'label', jsonb_build_object('zh', '意图', 'en', 'Intent'), 'weight', 0.3, 'instruction', COALESCE(abs.trigger_intent, 'Clarify the requested skill intent'), 'required_context', '[]'::jsonb, 'permission_tags', '[]'::jsonb),
        jsonb_build_object('key', 'context', 'label', jsonb_build_object('zh', '上下文', 'en', 'Context'), 'weight', 0.4, 'instruction', 'Collect governed business context through context rules', 'required_context', jsonb_build_array(COALESCE(NULLIF(abs.target_type, ''), 'target')), 'permission_tags', '[]'::jsonb),
        jsonb_build_object('key', 'action', 'label', jsonb_build_object('zh', '动作', 'en', 'Action'), 'weight', 0.3, 'instruction', COALESCE(abs.prompt_template, 'Execute the skill prompt'), 'required_context', '[]'::jsonb, 'permission_tags', COALESCE(abs.tool_allowlist, '[]'::jsonb))
    ),
    jsonb_build_object('source', 'legacy_assistant_business_skills', 'field_permission_catalog', true),
    jsonb_build_object('source', 'legacy_assistant_business_skills', 'context_engine_required', true),
    '{}'::jsonb,
    jsonb_build_object('saas_global', 'platform_admin', 'organization', 'organization_admin', 'deployment', 'deployment_admin'),
    COALESCE(abs.version, 1),
    COALESCE(NULLIF(abs.status, ''), 'draft'),
    abs.created_by,
    COALESCE(abs.created_by_type, ''),
    abs.reviewed_by,
    abs.source_session_id,
    COALESCE(abs.metadata, '{}'::jsonb) || jsonb_build_object('migrated_from', 'assistant_business_skills'),
    abs.created_at,
    abs.updated_at
FROM assistant_business_skills abs
ON CONFLICT (id) DO NOTHING;

DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'assistant_skill_runs'::regclass
      AND contype = 'f'
      AND pg_get_constraintdef(oid) LIKE '%skill_id%';

    IF constraint_name IS NOT NULL THEN
        EXECUTE FORMAT('ALTER TABLE assistant_skill_runs DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

ALTER TABLE assistant_skill_runs
    ADD CONSTRAINT assistant_skill_runs_skill_id_fkey
    FOREIGN KEY (skill_id) REFERENCES skill(id) ON DELETE CASCADE;

INSERT INTO data_table_catalog(
    table_name, master_table_name, detail_table_name, key_prefix, display_name,
    category, is_base_data, is_business_scenario, metadata
)
VALUES (
    'skill', 'skill_masters', 'skill_details', 'SKL', 'Skill',
    'ai', false, true, '{"unified_skill_table":true}'::jsonb
)
ON CONFLICT (table_name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    category = EXCLUDED.category,
    is_base_data = EXCLUDED.is_base_data,
    is_business_scenario = EXCLUDED.is_business_scenario,
    metadata = data_table_catalog.metadata || EXCLUDED.metadata,
    updated_at = NOW();

ALTER TABLE skill ADD COLUMN IF NOT EXISTS legacy_id UUID;
UPDATE skill SET legacy_id = id WHERE legacy_id IS NULL;
ALTER TABLE skill ADD COLUMN IF NOT EXISTS master_key TEXT;
ALTER TABLE skill ADD COLUMN IF NOT EXISTS parent_master_table TEXT;
ALTER TABLE skill ADD COLUMN IF NOT EXISTS parent_master_key TEXT;
UPDATE skill SET master_key = next_business_key('skill', 'SKL') WHERE master_key IS NULL;
ALTER TABLE skill ALTER COLUMN master_key SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_skill_master_key ON skill(master_key);
CREATE INDEX IF NOT EXISTS idx_skill_parent_master ON skill(parent_master_table, parent_master_key);

CREATE TABLE IF NOT EXISTS skill_details (
    sub_key TEXT PRIMARY KEY DEFAULT next_business_key('skill_details', 'SKLD'),
    master_key TEXT NOT NULL REFERENCES skill(master_key) ON DELETE CASCADE,
    parent_master_table TEXT,
    parent_master_key TEXT,
    detail_type TEXT NOT NULL DEFAULT 'field',
    line_no INT NOT NULL DEFAULT 0,
    field_key TEXT NOT NULL DEFAULT '',
    field_value JSONB NOT NULL DEFAULT 'null'::jsonb,
    payload JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_skill_details_master ON skill_details(master_key, line_no);

INSERT INTO data_field_catalog(table_name, field_name, data_type, display_name, is_master_key, is_visible_default, permission_level, display_order, metadata)
VALUES
    ('skill', 'id', 'uuid', 'ID', false, true, 'L1', 10, '{}'),
    ('skill', 'legacy_id', 'uuid', 'Legacy ID', false, false, 'L2', 15, '{}'),
    ('skill', 'master_key', 'text', 'Master Key', true, true, 'L1', 18, '{}'),
    ('skill', 'parent_master_table', 'text', 'Parent Master Table', false, false, 'L2', 19, '{}'),
    ('skill', 'parent_master_key', 'text', 'Parent Master Key', false, false, 'L2', 20, '{}'),
    ('skill', 'skill_key', 'text', 'Skill Key', false, true, 'L1', 30, '{}'),
    ('skill', 'scope_level', 'text', 'Scope Level', false, true, 'L2', 40, '{}'),
    ('skill', 'deployment_mode', 'text', 'Deployment Mode', false, true, 'L2', 50, '{}'),
    ('skill', 'organization_id', 'uuid', 'Organization ID', false, true, 'L2', 60, '{}'),
    ('skill', 'owner_user_id', 'uuid', 'Owner User ID', false, true, 'L2', 70, '{}'),
    ('skill', 'module_key', 'text', 'Module Key', false, true, 'L1', 80, '{}'),
    ('skill', 'target_type', 'text', 'Target Type', false, true, 'L1', 90, '{}'),
    ('skill', 'business_function_key', 'text', 'Business Function Key', false, true, 'L1', 100, '{}'),
    ('skill', 'name', 'text', 'Name', false, true, 'L1', 110, '{}'),
    ('skill', 'description', 'text', 'Description', false, true, 'L1', 120, '{}'),
    ('skill', 'trigger_intent', 'text', 'Trigger Intent', false, true, 'L2', 130, '{}'),
    ('skill', 'prompt_template', 'text', 'Prompt Template', false, false, 'L3', 140, '{"sensitive":true}'),
    ('skill', 'tool_allowlist', 'jsonb', 'Tool Allowlist', false, true, 'L3', 150, '{}'),
    ('skill', 'input_schema', 'jsonb', 'Input Schema', false, true, 'L2', 160, '{}'),
    ('skill', 'output_schema', 'jsonb', 'Output Schema', false, true, 'L2', 170, '{}'),
    ('skill', 'skill_components', 'jsonb', 'Skill Components', false, true, 'L2', 180, '{"component_count":"3-9"}'),
    ('skill', 'permission_policy', 'jsonb', 'Permission Policy', false, false, 'L3', 190, '{"sensitive":true}'),
    ('skill', 'context_policy', 'jsonb', 'Context Policy', false, false, 'L3', 200, '{"sensitive":true}'),
    ('skill', 'pricing_policy', 'jsonb', 'Pricing Policy', false, false, 'L3', 210, '{"sensitive":true}'),
    ('skill', 'activation_policy', 'jsonb', 'Activation Policy', false, false, 'L3', 220, '{"sensitive":true}'),
    ('skill', 'version', 'integer', 'Version', false, true, 'L1', 230, '{}'),
    ('skill', 'status', 'text', 'Status', false, true, 'L1', 240, '{}'),
    ('skill', 'metadata', 'jsonb', 'Metadata', false, false, 'L3', 250, '{"sensitive":true}')
ON CONFLICT (table_name, field_name) DO UPDATE SET
    data_type = EXCLUDED.data_type,
    display_name = EXCLUDED.display_name,
    is_master_key = EXCLUDED.is_master_key,
    is_visible_default = EXCLUDED.is_visible_default,
    permission_level = EXCLUDED.permission_level,
    display_order = EXCLUDED.display_order,
    metadata = data_field_catalog.metadata || EXCLUDED.metadata,
    updated_at = NOW();

INSERT INTO field_permission_rules(table_name, field_name, action, behavior, required_level, reason, metadata)
VALUES
    ('skill', 'prompt_template', 'read', 'approve', 'L3', 'skill prompt controls model behavior and requires governed access', '{"unified_skill":true}'::jsonb),
    ('skill', 'permission_policy', 'read', 'approve', 'L3', 'skill permission policy is sensitive governance data', '{"unified_skill":true}'::jsonb),
    ('skill', 'context_policy', 'read', 'approve', 'L3', 'skill context policy controls business context admission', '{"unified_skill":true}'::jsonb),
    ('skill', 'pricing_policy', 'read', 'approve', 'L3', 'skill pricing policy affects commercial behavior', '{"unified_skill":true}'::jsonb),
    ('skill', 'activation_policy', 'write', 'approve', 'L3', 'skill activation policy changes approval authority', '{"unified_skill":true}'::jsonb),
    ('skill', 'metadata', 'read', 'approve', 'L3', 'skill metadata can contain imported governance details', '{"unified_skill":true}'::jsonb);

WITH skill_dictionary AS (
    INSERT INTO context_dictionary_versions(scope_level, module_key, version_key, source_type, source_name, status, metadata)
    VALUES ('saas', 'assistant', 'unified-skill-v1', 'json', 'migration_036_unified_skill', 'active', '{"entity":"skill"}'::jsonb)
    ON CONFLICT (scope_level, organization_id, module_key, version_key) DO UPDATE
    SET status = 'active', updated_at = NOW()
    RETURNING id
),
skill_domain AS (
    INSERT INTO context_business_domains(dictionary_version_id, module_key, name, scope_level, status, metadata)
    SELECT id, 'assistant', 'Assistant Skill', 'saas', 'active', '{"unified_skill":true}'::jsonb
    FROM skill_dictionary
    ON CONFLICT (dictionary_version_id, module_key) DO UPDATE
    SET status = 'active'
    RETURNING id, dictionary_version_id
),
skill_entity AS (
    INSERT INTO context_entities(dictionary_version_id, domain_id, entity_key, display_name, description, status, metadata)
    SELECT dictionary_version_id, id, 'skill', 'Skill', 'Unified governed skill table', 'active', '{"table_name":"skill"}'::jsonb
    FROM skill_domain
    ON CONFLICT (dictionary_version_id, entity_key) DO UPDATE
    SET status = 'active', display_name = EXCLUDED.display_name, description = EXCLUDED.description
    RETURNING id, dictionary_version_id
),
skill_fields AS (
    INSERT INTO context_fields(dictionary_version_id, entity_id, field_key, display_name, data_type, semantic_type, sensitivity_level, base_weight, is_governance_field, mask_strategy, status, metadata)
    SELECT dictionary_version_id, id, 'skill_components', 'Skill Components', 'jsonb', 'skill_component_weights', 'normal', 9, true, 'none', 'active', '{"component_count":"3-9"}'::jsonb FROM skill_entity
    UNION ALL SELECT dictionary_version_id, id, 'permission_policy', 'Permission Policy', 'jsonb', 'permission_policy', 'restricted', 8, true, 'summary', 'active', '{}'::jsonb FROM skill_entity
    UNION ALL SELECT dictionary_version_id, id, 'context_policy', 'Context Policy', 'jsonb', 'context_policy', 'restricted', 8, true, 'summary', 'active', '{}'::jsonb FROM skill_entity
    UNION ALL SELECT dictionary_version_id, id, 'pricing_policy', 'Pricing Policy', 'jsonb', 'pricing_policy', 'sensitive', 6, false, 'summary', 'active', '{}'::jsonb FROM skill_entity
    ON CONFLICT (entity_id, field_key) DO UPDATE
    SET status = 'active', display_name = EXCLUDED.display_name, sensitivity_level = EXCLUDED.sensitivity_level, base_weight = EXCLUDED.base_weight
    RETURNING id, dictionary_version_id, entity_id, field_key
)
INSERT INTO context_physical_mappings(dictionary_version_id, entity_id, field_id, table_name, column_name, tenant_column, status, metadata)
SELECT dictionary_version_id, entity_id, id, 'skill', field_key, 'organization_id', 'active', '{"unified_skill":true}'::jsonb
FROM skill_fields;
