-- 026_master_detail_permissions.sql
-- Compatible foundation for physical master/detail records, user field preferences,
-- and table/field level authorization.

CREATE TABLE IF NOT EXISTS data_table_catalog (
    table_name           TEXT PRIMARY KEY,
    master_table_name    TEXT NOT NULL,
    detail_table_name    TEXT NOT NULL,
    key_prefix           TEXT NOT NULL,
    display_name         TEXT NOT NULL DEFAULT '',
    category             TEXT NOT NULL DEFAULT 'system',
    is_base_data         BOOLEAN NOT NULL DEFAULT false,
    is_business_scenario BOOLEAN NOT NULL DEFAULT false,
    metadata             JSONB NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS data_field_catalog (
    table_name         TEXT NOT NULL REFERENCES data_table_catalog(table_name) ON DELETE CASCADE,
    field_name         TEXT NOT NULL,
    data_type          TEXT NOT NULL DEFAULT '',
    display_name       TEXT NOT NULL DEFAULT '',
    is_master_key      BOOLEAN NOT NULL DEFAULT false,
    is_sub_key         BOOLEAN NOT NULL DEFAULT false,
    is_visible_default BOOLEAN NOT NULL DEFAULT true,
    permission_level   TEXT NOT NULL DEFAULT 'L1'
        CHECK (permission_level IN ('L1', 'L2', 'L3', 'L4')),
    display_order      INT NOT NULL DEFAULT 0,
    metadata           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (table_name, field_name)
);

CREATE TABLE IF NOT EXISTS user_field_preferences (
    actor_id       TEXT NOT NULL,
    table_name     TEXT NOT NULL REFERENCES data_table_catalog(table_name) ON DELETE CASCADE,
    visible_fields JSONB NOT NULL DEFAULT '[]',
    field_order    JSONB NOT NULL DEFAULT '[]',
    field_widths   JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (actor_id, table_name)
);

CREATE TABLE IF NOT EXISTS field_permission_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name       TEXT NOT NULL REFERENCES data_table_catalog(table_name) ON DELETE CASCADE,
    field_name       TEXT NOT NULL DEFAULT '*',
    actor_type       TEXT NOT NULL DEFAULT '*',
    actor_id         TEXT,
    role_id          UUID REFERENCES roles(id) ON DELETE CASCADE,
    action           TEXT NOT NULL CHECK (action IN ('read', 'write', 'delete', 'admin')),
    behavior         TEXT NOT NULL DEFAULT 'allow'
        CHECK (behavior IN ('allow', 'notify', 'approve', 'deny')),
    required_level   TEXT NOT NULL DEFAULT 'L1'
        CHECK (required_level IN ('L1', 'L2', 'L3', 'L4')),
    reason           TEXT NOT NULL DEFAULT '',
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_field_permission_rules_lookup
    ON field_permission_rules(table_name, field_name, action, actor_type, actor_id);

CREATE TABLE IF NOT EXISTS business_key_sequences (
    entity_name TEXT NOT NULL,
    date_key    DATE NOT NULL,
    next_value  BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (entity_name, date_key)
);

CREATE OR REPLACE FUNCTION next_business_key(p_entity_name TEXT, p_prefix TEXT DEFAULT NULL)
RETURNS TEXT
LANGUAGE plpgsql
AS $$
DECLARE
    v_date DATE := CURRENT_DATE;
    v_value BIGINT;
    v_prefix TEXT;
BEGIN
    SELECT key_prefix INTO v_prefix
    FROM data_table_catalog
    WHERE table_name = p_entity_name;

    v_prefix := COALESCE(
        NULLIF(p_prefix, ''),
        NULLIF(v_prefix, ''),
        UPPER(LEFT(REGEXP_REPLACE(p_entity_name, '[^A-Za-z0-9]', '', 'g'), 6))
    );

    INSERT INTO business_key_sequences(entity_name, date_key, next_value)
    VALUES (p_entity_name, v_date, 2)
    ON CONFLICT (entity_name, date_key)
    DO UPDATE SET next_value = business_key_sequences.next_value + 1
    RETURNING next_value - 1 INTO v_value;

    RETURN v_prefix || '-' || TO_CHAR(v_date, 'YYYYMMDD') || '-' || LPAD(v_value::TEXT, 6, '0');
END;
$$;

DO $$
DECLARE
    rec RECORD;
    v_detail_table TEXT;
    v_prefix TEXT;
    v_detail_prefix TEXT;
    v_has_uuid_id BOOLEAN;
BEGIN
    FOR rec IN
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_type = 'BASE TABLE'
          AND table_name NOT IN (
              'schema_migrations',
              'data_table_catalog',
              'data_field_catalog',
              'user_field_preferences',
              'field_permission_rules',
              'business_key_sequences'
          )
          AND table_name NOT LIKE '%\_details' ESCAPE '\'
        ORDER BY table_name
    LOOP
        v_detail_table := rec.table_name || '_details';
        v_prefix := UPPER(LEFT(REGEXP_REPLACE(rec.table_name, '[^A-Za-z0-9]', '', 'g'), 6));
        v_detail_prefix := UPPER(LEFT(REGEXP_REPLACE(v_detail_table, '[^A-Za-z0-9]', '', 'g'), 6));

        INSERT INTO data_table_catalog(
            table_name,
            master_table_name,
            detail_table_name,
            key_prefix,
            display_name,
            category,
            is_base_data,
            is_business_scenario
        )
        VALUES (
            rec.table_name,
            rec.table_name || '_masters',
            v_detail_table,
            v_prefix,
            rec.table_name,
            CASE
                WHEN rec.table_name IN (
                    'requirements', 'requirement_documents', 'requirement_analysis_workflows',
                    'projects', 'project_members', 'project_workflows', 'deliverables',
                    'project_cost_entries', 'project_evaluations', 'meta_resources',
                    'demand_profiles', 'pdca_cycles', 'pdca_events', 'finance_export_batches',
                    'finance_export_lines', 'tool_executions', 'tool_approvals'
                ) THEN 'business'
                ELSE 'base_data'
            END,
            rec.table_name NOT IN (
                'requirements', 'requirement_documents', 'requirement_analysis_workflows',
                'projects', 'project_members', 'project_workflows', 'deliverables',
                'project_cost_entries', 'project_evaluations', 'meta_resources',
                'demand_profiles', 'pdca_cycles', 'pdca_events', 'finance_export_batches',
                'finance_export_lines', 'tool_executions', 'tool_approvals',
                'traces', 'spans', 'metrics', 'assistant_sessions', 'assistant_messages',
                'assistant_steps', 'assistant_memories', 'ai_invocations', 'ai_usage_ledger',
                'access_decisions', 'capability_invocations', 'finance_webhook_events'
            ),
            rec.table_name IN (
                'requirements', 'requirement_documents', 'requirement_analysis_workflows',
                'projects', 'project_members', 'project_workflows', 'deliverables',
                'project_cost_entries', 'project_evaluations', 'meta_resources',
                'demand_profiles', 'pdca_cycles', 'pdca_events', 'finance_export_batches',
                'finance_export_lines', 'tool_executions', 'tool_approvals'
            )
        )
        ON CONFLICT (table_name) DO UPDATE SET
            master_table_name = EXCLUDED.master_table_name,
            detail_table_name = EXCLUDED.detail_table_name,
            key_prefix = EXCLUDED.key_prefix,
            category = EXCLUDED.category,
            is_base_data = EXCLUDED.is_base_data,
            is_business_scenario = EXCLUDED.is_business_scenario,
            updated_at = NOW();

        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = rec.table_name
              AND column_name = 'id'
              AND data_type = 'uuid'
        ) INTO v_has_uuid_id;

        EXECUTE FORMAT('ALTER TABLE %I ADD COLUMN IF NOT EXISTS legacy_id UUID', rec.table_name);
        IF v_has_uuid_id THEN
            EXECUTE FORMAT('UPDATE %I SET legacy_id = id WHERE legacy_id IS NULL AND id IS NOT NULL', rec.table_name);
        END IF;
        EXECUTE FORMAT('ALTER TABLE %I ADD COLUMN IF NOT EXISTS master_key TEXT', rec.table_name);
        EXECUTE FORMAT('ALTER TABLE %I ADD COLUMN IF NOT EXISTS parent_master_table TEXT', rec.table_name);
        EXECUTE FORMAT('ALTER TABLE %I ADD COLUMN IF NOT EXISTS parent_master_key TEXT', rec.table_name);
        EXECUTE FORMAT('UPDATE %I SET master_key = next_business_key(%L, %L) WHERE master_key IS NULL', rec.table_name, rec.table_name, v_prefix);
        EXECUTE FORMAT('ALTER TABLE %I ALTER COLUMN master_key SET NOT NULL', rec.table_name);
        EXECUTE FORMAT('CREATE UNIQUE INDEX IF NOT EXISTS %I ON %I(master_key)', 'uq_' || rec.table_name || '_master_key', rec.table_name);
        EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(parent_master_table, parent_master_key)', 'idx_' || rec.table_name || '_parent_master', rec.table_name);

        EXECUTE FORMAT(
            'CREATE TABLE IF NOT EXISTS %I (
                sub_key TEXT PRIMARY KEY DEFAULT next_business_key(%L, %L),
                master_key TEXT NOT NULL REFERENCES %I(master_key) ON DELETE CASCADE,
                parent_master_table TEXT,
                parent_master_key TEXT,
                detail_type TEXT NOT NULL DEFAULT ''field'',
                line_no INT NOT NULL DEFAULT 0,
                field_key TEXT NOT NULL DEFAULT '''',
                field_value JSONB NOT NULL DEFAULT ''null''::jsonb,
                payload JSONB NOT NULL DEFAULT ''{}''::jsonb,
                metadata JSONB NOT NULL DEFAULT ''{}''::jsonb,
                created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )',
            v_detail_table, v_detail_table, v_detail_prefix, rec.table_name
        );
        EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(master_key, line_no)', 'idx_' || v_detail_table || '_master', v_detail_table);
    END LOOP;
END;
$$;

INSERT INTO data_field_catalog(table_name, field_name, data_type, display_name, is_master_key, is_visible_default, display_order)
SELECT
    c.table_name,
    c.column_name,
    c.data_type,
    c.column_name,
    c.column_name = 'master_key',
    c.column_name NOT IN ('api_key_hash', 'password_hash', 'secret_ciphertext', 'content', 'metadata'),
    c.ordinal_position
FROM information_schema.columns c
JOIN data_table_catalog t ON t.table_name = c.table_name
WHERE c.table_schema = 'public'
ON CONFLICT (table_name, field_name) DO UPDATE SET
    data_type = EXCLUDED.data_type,
    is_master_key = EXCLUDED.is_master_key,
    is_visible_default = EXCLUDED.is_visible_default,
    display_order = EXCLUDED.display_order,
    updated_at = NOW();

INSERT INTO field_permission_rules(table_name, field_name, action, behavior, required_level, reason)
SELECT table_name, field_name, 'read', 'deny', 'L4', 'sensitive field is hidden by default'
FROM data_field_catalog
WHERE field_name IN ('api_key_hash', 'password_hash', 'secret_ciphertext', 'content')
ON CONFLICT DO NOTHING;
