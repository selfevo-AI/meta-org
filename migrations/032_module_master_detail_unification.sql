-- 032_module_master_detail_unification.sql
-- Canonical module-level master/detail tables.
--
-- This migration is intentionally non-destructive. Legacy tables remain in place
-- until migration validation passes and a separate, explicit drop migration is
-- approved.

CREATE TABLE IF NOT EXISTS module_master_source_catalog (
    module_name     TEXT NOT NULL,
    source_table    TEXT PRIMARY KEY,
    entity_type     TEXT NOT NULL,
    relation_mode   TEXT NOT NULL DEFAULT 'master'
        CHECK (relation_mode IN ('master', 'detail')),
    parent_table    TEXT,
    parent_fk       TEXT,
    key_prefix      TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO module_master_source_catalog(module_name, source_table, entity_type, relation_mode, parent_table, parent_fk, key_prefix)
VALUES
    ('identity', 'users', 'user', 'master', NULL, NULL, 'USR'),
    ('identity', 'ai_agents', 'ai_agent', 'master', NULL, NULL, 'AGT'),
    ('identity', 'roles', 'role', 'master', NULL, NULL, 'ROL'),
    ('identity', 'user_roles', 'user_role', 'detail', 'users', 'user_id', 'URO'),
    ('identity', 'agent_roles', 'agent_role', 'detail', 'ai_agents', 'agent_id', 'ARO'),

    ('organization', 'organizations', 'organization', 'master', NULL, NULL, 'ORG'),
    ('organization', 'muvrs', 'mvru', 'detail', 'organizations', 'organization_id', 'MVR'),
    ('organization', 'teams', 'team', 'detail', 'muvrs', 'mvru_id', 'TEM'),
    ('organization', 'mvru_members', 'mvru_member', 'detail', 'muvrs', 'mvru_id', 'MMM'),
    ('organization', 'mvru_relationships', 'mvru_relationship', 'detail', 'muvrs', 'source_mvru_id', 'MRL'),
    ('organization', 'departments', 'department', 'detail', 'organizations', 'organization_id', 'DEP'),
    ('organization', 'external_members', 'external_member', 'master', NULL, NULL, 'EXT'),
    ('organization', 'organization_memberships', 'organization_membership', 'detail', 'departments', 'department_id', 'OMB'),
    ('organization', 'department_mvru_links', 'department_mvru_link', 'detail', 'departments', 'department_id', 'DML'),
    ('organization', 'positions', 'position', 'detail', 'departments', 'department_id', 'POS'),
    ('organization', 'position_assignments', 'position_assignment', 'detail', 'positions', 'position_id', 'PAS'),
    ('organization', 'employee_profiles', 'employee_profile', 'detail', 'users', 'user_id', 'EMP'),

    ('layer', 'layer_configs', 'layer_config', 'detail', 'muvrs', 'mvru_id', 'LAY'),
    ('layer', 'layer_routing_rules', 'layer_routing_rule', 'detail', 'muvrs', 'mvru_id', 'LRR'),

    ('capability', 'capabilities', 'capability', 'master', NULL, NULL, 'CAP'),
    ('capability', 'capability_bindings', 'capability_binding', 'detail', 'capabilities', 'capability_id', 'CBN'),
    ('capability', 'capability_invocations', 'capability_invocation', 'detail', 'capabilities', 'capability_id', 'CIN'),
    ('capability', 'capability_evaluations', 'capability_evaluation', 'detail', 'capabilities', 'capability_id', 'CEV'),

    ('workflow', 'workflow_templates', 'workflow_template', 'master', NULL, NULL, 'WFT'),
    ('workflow', 'workflow_instances', 'workflow_instance', 'detail', 'workflow_templates', 'template_id', 'WFI'),
    ('workflow', 'tasks', 'task', 'detail', 'workflow_instances', 'workflow_id', 'TSK'),
    ('workflow', 'decisions', 'decision', 'detail', 'tasks', 'task_id', 'DEC'),
    ('workflow', 'workflow_contexts', 'workflow_context', 'detail', 'workflow_instances', 'workflow_id', 'WFC'),
    ('workflow', 'task_matrix_assignments', 'task_matrix_assignment', 'detail', 'tasks', 'task_id', 'TMA'),

    ('project_lifecycle', 'requirements', 'requirement', 'master', NULL, NULL, 'REQ'),
    ('project_lifecycle', 'requirement_documents', 'requirement_document', 'detail', 'requirements', 'requirement_id', 'RDOC'),
    ('project_lifecycle', 'requirement_analysis_workflows', 'requirement_analysis_workflow', 'detail', 'requirements', 'requirement_id', 'RAW'),
    ('project_lifecycle', 'projects', 'project', 'detail', 'requirements', 'requirement_id', 'PRJ'),
    ('project_lifecycle', 'project_members', 'project_member', 'detail', 'projects', 'project_id', 'PMB'),
    ('project_lifecycle', 'project_workflows', 'project_workflow', 'detail', 'projects', 'project_id', 'PWF'),
    ('project_lifecycle', 'deliverables', 'deliverable', 'detail', 'projects', 'project_id', 'DEL'),
    ('project_lifecycle', 'project_cost_entries', 'project_cost_entry', 'detail', 'projects', 'project_id', 'PCE'),
    ('project_lifecycle', 'project_evaluations', 'project_evaluation', 'detail', 'projects', 'project_id', 'PEV'),

    ('finance', 'finance_adapters', 'finance_adapter', 'master', NULL, NULL, 'FAD'),
    ('finance', 'finance_export_batches', 'finance_export_batch', 'detail', 'finance_adapters', 'adapter_id', 'FEB'),
    ('finance', 'finance_export_lines', 'finance_export_line', 'detail', 'finance_export_batches', 'batch_id', 'FEL'),
    ('finance', 'finance_webhook_events', 'finance_webhook_event', 'detail', 'finance_adapters', 'adapter_id', 'FWE'),
    ('finance', 'finance_import_batches', 'finance_import_batch', 'detail', 'finance_adapters', 'adapter_id', 'FIB'),
    ('finance', 'finance_import_records', 'finance_import_record', 'detail', 'finance_import_batches', 'batch_id', 'FIR'),
    ('finance', 'finance_payables', 'finance_payable', 'master', NULL, NULL, 'FPY'),
    ('finance', 'finance_payments', 'finance_payment', 'master', NULL, NULL, 'FPM'),
    ('finance', 'finance_payment_allocations', 'finance_payment_allocation', 'detail', 'finance_payments', 'payment_id', 'FPA'),
    ('finance', 'finance_settlement_orders', 'finance_settlement_order', 'master', NULL, NULL, 'FSO'),
    ('finance', 'finance_settlement_lines', 'finance_settlement_line', 'detail', 'finance_settlement_orders', 'settlement_order_id', 'FSL'),
    ('finance', 'finance_receivables', 'finance_receivable', 'detail', 'finance_settlement_orders', 'settlement_order_id', 'FRC'),
    ('finance', 'finance_receivable_lines', 'finance_receivable_line', 'detail', 'finance_receivables', 'receivable_id', 'FRL'),
    ('finance', 'finance_receipts', 'finance_receipt', 'master', NULL, NULL, 'FRP'),
    ('finance', 'finance_receipt_allocations', 'finance_receipt_allocation', 'detail', 'finance_receipts', 'receipt_id', 'FRA'),

    ('costing', 'currencies', 'currency', 'master', NULL, NULL, 'CUR'),
    ('costing', 'exchange_rate_versions', 'exchange_rate_version', 'detail', 'currencies', 'from_currency', 'ERV'),
    ('costing', 'cost_rate_cards', 'cost_rate_card', 'detail', 'currencies', 'currency', 'CRC'),
    ('costing', 'cost_budgets', 'cost_budget', 'detail', 'currencies', 'currency', 'CBU'),
    ('costing', 'cost_ledger_entries', 'cost_ledger_entry', 'detail', 'currencies', 'currency', 'CLE'),

    ('aigateway', 'model_providers', 'model_provider', 'master', NULL, NULL, 'AIP'),
    ('aigateway', 'models', 'model', 'detail', 'model_providers', 'provider_id', 'AIM'),
    ('aigateway', 'model_price_versions', 'model_price_version', 'detail', 'models', 'model_id', 'MPV'),
    ('aigateway', 'ai_invocations', 'ai_invocation', 'detail', 'models', 'model_id', 'AIN'),
    ('aigateway', 'ai_usage_ledger', 'ai_usage_ledger', 'detail', 'ai_invocations', 'invocation_id', 'AUL'),
    ('aigateway', 'model_provider_channels', 'model_provider_channel', 'detail', 'model_providers', 'provider_id', 'MPC'),
    ('aigateway', 'ai_routing_rules', 'ai_routing_rule', 'detail', 'model_providers', 'provider_id', 'ARR'),

    ('toolruntime', 'tool_definitions', 'tool_definition', 'master', NULL, NULL, 'TLD'),
    ('toolruntime', 'interface_files', 'interface_file', 'master', NULL, NULL, 'IFL'),
    ('toolruntime', 'tool_executions', 'tool_execution', 'detail', 'tool_definitions', 'tool_id', 'TEX'),
    ('toolruntime', 'tool_approvals', 'tool_approval', 'detail', 'tool_executions', 'execution_id', 'TAP'),

    ('assistant', 'assistant_sessions', 'assistant_session', 'master', NULL, NULL, 'ASN'),
    ('assistant', 'assistant_messages', 'assistant_message', 'detail', 'assistant_sessions', 'session_id', 'AMG'),
    ('assistant', 'assistant_steps', 'assistant_step', 'detail', 'assistant_sessions', 'session_id', 'AST'),
    ('assistant', 'assistant_memories', 'assistant_memory', 'master', NULL, NULL, 'AMR'),
    ('assistant', 'assistant_module_defaults', 'assistant_module_default', 'master', NULL, NULL, 'AMD'),
    ('assistant', 'assistant_proposals', 'assistant_proposal', 'detail', 'assistant_sessions', 'session_id', 'APR'),
    ('assistant', 'assistant_business_skills', 'assistant_business_skill', 'master', NULL, NULL, 'ABS'),
    ('assistant', 'assistant_skill_runs', 'assistant_skill_run', 'detail', 'assistant_business_skills', 'skill_id', 'ASR'),

    ('governance', 'permissions', 'permission', 'master', NULL, NULL, 'PER'),
    ('governance', 'principles', 'principle', 'master', NULL, NULL, 'PRI'),
    ('governance', 'control_rules', 'control_rule', 'detail', 'principles', 'principle_id', 'CRL'),
    ('governance', 'access_decisions', 'access_decision', 'master', NULL, NULL, 'ACD'),
    ('governance', 'context_weight_scores', 'context_weight_score', 'detail', 'workflow_templates', 'workflow_template_id', 'CWS'),
    ('governance', 'field_permission_rules', 'field_permission_rule', 'master', NULL, NULL, 'FPR'),
    ('governance', 'user_field_preferences', 'user_field_preference', 'master', NULL, NULL, 'UFP'),

    ('verification', 'verification_reports', 'verification_report', 'master', NULL, NULL, 'VRP'),
    ('verification', 'review_assignments', 'review_assignment', 'detail', 'verification_reports', 'report_id', 'RVA'),

    ('evolution', 'weight_scores', 'weight_score', 'master', NULL, NULL, 'WSC'),
    ('evolution', 'weight_alphas', 'weight_alpha', 'master', NULL, NULL, 'WAL'),
    ('evolution', 'experiments', 'experiment', 'master', NULL, NULL, 'EXP'),
    ('evolution', 'knowledge_entries', 'knowledge_entry', 'master', NULL, NULL, 'KNE'),
    ('evolution', 'signals', 'signal', 'master', NULL, NULL, 'SIG'),

    ('observability', 'traces', 'trace', 'master', NULL, NULL, 'TRC'),
    ('observability', 'spans', 'span', 'detail', 'traces', 'trace_id', 'SPN'),
    ('observability', 'metrics', 'metric', 'master', NULL, NULL, 'MET'),

    ('metaresource', 'meta_resources', 'meta_resource', 'master', NULL, NULL, 'MRS'),
    ('metaresource', 'demand_profiles', 'demand_profile', 'detail', 'requirements', 'requirement_id', 'DPR'),
    ('metaresource', 'pdca_cycles', 'pdca_cycle', 'detail', 'demand_profiles', 'demand_profile_id', 'PDC'),
    ('metaresource', 'pdca_events', 'pdca_event', 'detail', 'pdca_cycles', 'cycle_id', 'PDE')
ON CONFLICT (source_table) DO UPDATE SET
    module_name = EXCLUDED.module_name,
    entity_type = EXCLUDED.entity_type,
    relation_mode = EXCLUDED.relation_mode,
    parent_table = EXCLUDED.parent_table,
    parent_fk = EXCLUDED.parent_fk,
    key_prefix = EXCLUDED.key_prefix,
    updated_at = NOW();

CREATE OR REPLACE FUNCTION module_table_exists(p_table TEXT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = p_table
          AND table_type = 'BASE TABLE'
    );
$$;

CREATE OR REPLACE FUNCTION module_column_exists(p_table TEXT, p_column TEXT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = p_table
          AND column_name = p_column
    );
$$;

CREATE OR REPLACE FUNCTION module_column_udt(p_table TEXT, p_column TEXT)
RETURNS TEXT
LANGUAGE sql
STABLE
AS $$
    SELECT udt_name
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = p_table
      AND column_name = p_column
    LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION module_first_text_expr(p_table TEXT, p_columns TEXT[], p_alias TEXT)
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    v_column TEXT;
BEGIN
    FOREACH v_column IN ARRAY p_columns LOOP
        IF module_column_exists(p_table, v_column) THEN
            RETURN FORMAT('COALESCE(%s.%I::TEXT, '''')', p_alias, v_column);
        END IF;
    END LOOP;
    RETURN '''''';
END;
$$;

CREATE OR REPLACE FUNCTION module_uuid_expr(p_table TEXT, p_column TEXT, p_alias TEXT)
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    IF module_column_exists(p_table, p_column) AND module_column_udt(p_table, p_column) = 'uuid' THEN
        RETURN FORMAT('%s.%I', p_alias, p_column);
    END IF;
    RETURN 'NULL::UUID';
END;
$$;

CREATE OR REPLACE FUNCTION module_jsonb_expr(p_table TEXT, p_column TEXT, p_alias TEXT)
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    IF module_column_exists(p_table, p_column) AND module_column_udt(p_table, p_column) IN ('json', 'jsonb') THEN
        RETURN FORMAT('COALESCE(%s.%I::JSONB, ''{}''::JSONB)', p_alias, p_column);
    END IF;
    RETURN '''{}''::JSONB';
END;
$$;

CREATE OR REPLACE FUNCTION module_timestamp_expr(p_table TEXT, p_column TEXT, p_alias TEXT)
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    IF module_column_exists(p_table, p_column) THEN
        RETURN FORMAT('COALESCE(%s.%I::TIMESTAMPTZ, NOW())', p_alias, p_column);
    END IF;
    IF p_column = 'updated_at' AND module_column_exists(p_table, 'created_at') THEN
        RETURN FORMAT('COALESCE(%s.created_at::TIMESTAMPTZ, NOW())', p_alias);
    END IF;
    RETURN 'NOW()';
END;
$$;

CREATE OR REPLACE FUNCTION module_legacy_pk_expr(p_table TEXT, p_alias TEXT)
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    IF module_column_exists(p_table, 'id') THEN
        RETURN FORMAT('%s.id::TEXT', p_alias);
    ELSIF module_column_exists(p_table, 'code') THEN
        RETURN FORMAT('%s.code::TEXT', p_alias);
    ELSIF module_column_exists(p_table, 'user_id') THEN
        RETURN FORMAT('%s.user_id::TEXT', p_alias);
    ELSIF module_column_exists(p_table, 'entity_name') THEN
        RETURN FORMAT('%s.entity_name::TEXT', p_alias);
    END IF;
    RETURN FORMAT('%s.master_key::TEXT', p_alias);
END;
$$;

CREATE OR REPLACE FUNCTION ensure_module_master_detail_tables(p_module_name TEXT, p_key_prefix TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    v_master_table TEXT := p_module_name || '_masters';
    v_detail_table TEXT := p_module_name || '_details';
    v_master_prefix TEXT := UPPER(LEFT(REGEXP_REPLACE(p_module_name, '[^A-Za-z0-9]', '', 'g'), 5)) || 'M';
    v_detail_prefix TEXT := UPPER(LEFT(REGEXP_REPLACE(p_module_name, '[^A-Za-z0-9]', '', 'g'), 5)) || 'D';
BEGIN
    EXECUTE FORMAT(
        'CREATE TABLE IF NOT EXISTS %I (
            master_key TEXT PRIMARY KEY DEFAULT next_business_key(%L, %L),
            entity_type TEXT NOT NULL,
            legacy_table TEXT NOT NULL,
            legacy_pk TEXT NOT NULL,
            legacy_id UUID,
            title TEXT NOT NULL DEFAULT '''',
            name TEXT NOT NULL DEFAULT '''',
            code TEXT NOT NULL DEFAULT '''',
            status TEXT NOT NULL DEFAULT '''',
            organization_id UUID,
            department_id UUID,
            project_id UUID,
            requirement_id UUID,
            workflow_id UUID,
            task_id UUID,
            actor_id UUID,
            actor_type TEXT NOT NULL DEFAULT '''',
            core_data JSONB NOT NULL DEFAULT ''{}''::JSONB,
            metadata JSONB NOT NULL DEFAULT ''{}''::JSONB,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            UNIQUE (legacy_table, legacy_pk)
        )',
        v_master_table,
        v_master_table,
        COALESCE(NULLIF(p_key_prefix, ''), v_master_prefix)
    );

    EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(entity_type, status)', 'idx_' || v_master_table || '_entity_status', v_master_table);
    EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(legacy_table, legacy_id)', 'idx_' || v_master_table || '_legacy_id', v_master_table);
    EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(organization_id, department_id)', 'idx_' || v_master_table || '_org', v_master_table);

    EXECUTE FORMAT(
        'CREATE TABLE IF NOT EXISTS %I (
            sub_key TEXT PRIMARY KEY DEFAULT next_business_key(%L, %L),
            master_key TEXT NOT NULL REFERENCES %I(master_key) ON DELETE CASCADE,
            source_master_key TEXT NOT NULL DEFAULT '''',
            detail_type TEXT NOT NULL,
            legacy_table TEXT NOT NULL,
            legacy_pk TEXT NOT NULL,
            legacy_id UUID,
            parent_legacy_table TEXT NOT NULL DEFAULT '''',
            parent_legacy_pk TEXT NOT NULL DEFAULT '''',
            parent_legacy_id UUID,
            line_no INT NOT NULL DEFAULT 0,
            field_key TEXT NOT NULL DEFAULT '''',
            payload JSONB NOT NULL DEFAULT ''{}''::JSONB,
            metadata JSONB NOT NULL DEFAULT ''{}''::JSONB,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            UNIQUE (legacy_table, legacy_pk)
        )',
        v_detail_table,
        v_detail_table,
        COALESCE(NULLIF(p_key_prefix, ''), v_detail_prefix),
        v_master_table
    );

    EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(master_key, line_no)', 'idx_' || v_detail_table || '_master_line', v_detail_table);
    EXECUTE FORMAT('CREATE INDEX IF NOT EXISTS %I ON %I(detail_type)', 'idx_' || v_detail_table || '_detail_type', v_detail_table);

    INSERT INTO data_table_catalog(table_name, master_table_name, detail_table_name, key_prefix, display_name, category, is_base_data, is_business_scenario, metadata)
    VALUES
        (v_master_table, v_master_table, v_detail_table, COALESCE(NULLIF(p_key_prefix, ''), v_master_prefix), v_master_table, 'canonical', false, true,
         jsonb_build_object('module_name', p_module_name, 'canonical_role', 'master')),
        (v_detail_table, v_master_table, v_detail_table, COALESCE(NULLIF(p_key_prefix, ''), v_detail_prefix), v_detail_table, 'canonical', false, true,
         jsonb_build_object('module_name', p_module_name, 'canonical_role', 'detail'))
    ON CONFLICT (table_name) DO UPDATE SET
        master_table_name = EXCLUDED.master_table_name,
        detail_table_name = EXCLUDED.detail_table_name,
        key_prefix = EXCLUDED.key_prefix,
        category = EXCLUDED.category,
        is_base_data = EXCLUDED.is_base_data,
        is_business_scenario = EXCLUDED.is_business_scenario,
        metadata = data_table_catalog.metadata || EXCLUDED.metadata,
        updated_at = NOW();
END;
$$;

DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT module_name, MIN(key_prefix) AS key_prefix
        FROM module_master_source_catalog
        GROUP BY module_name
        ORDER BY module_name
    LOOP
        PERFORM ensure_module_master_detail_tables(rec.module_name, rec.key_prefix);
    END LOOP;
END;
$$;

CREATE OR REPLACE FUNCTION ensure_source_master_key(p_source_table TEXT, p_key_prefix TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    v_has_uuid_id BOOLEAN;
BEGIN
    IF NOT module_table_exists(p_source_table) THEN
        RETURN;
    END IF;

    INSERT INTO data_table_catalog(table_name, master_table_name, detail_table_name, key_prefix, display_name, category, is_base_data, is_business_scenario, metadata)
    VALUES (
        p_source_table,
        p_source_table || '_masters',
        p_source_table || '_details',
        p_key_prefix,
        p_source_table,
        'legacy',
        false,
        false,
        jsonb_build_object('deprecated', true, 'canonicalized_by', '032_module_master_detail_unification.sql')
    )
    ON CONFLICT (table_name) DO UPDATE SET
        key_prefix = COALESCE(NULLIF(data_table_catalog.key_prefix, ''), EXCLUDED.key_prefix),
        category = 'legacy',
        metadata = data_table_catalog.metadata || EXCLUDED.metadata,
        updated_at = NOW();

    SELECT module_column_exists(p_source_table, 'id') AND module_column_udt(p_source_table, 'id') = 'uuid'
    INTO v_has_uuid_id;

    EXECUTE FORMAT('ALTER TABLE %I ADD COLUMN IF NOT EXISTS legacy_id UUID', p_source_table);
    IF v_has_uuid_id THEN
        EXECUTE FORMAT('UPDATE %I SET legacy_id = id WHERE legacy_id IS NULL AND id IS NOT NULL', p_source_table);
    END IF;

    EXECUTE FORMAT('ALTER TABLE %I ADD COLUMN IF NOT EXISTS master_key TEXT', p_source_table);
    EXECUTE FORMAT(
        'UPDATE %I
         SET master_key = next_business_key(%L, %L)
         WHERE master_key IS NULL
            OR (COALESCE($1, '''') <> '''' AND master_key NOT LIKE COALESCE($1, '''') || ''-%%'')',
        p_source_table,
        p_source_table,
        p_key_prefix
    )
    USING p_key_prefix;
    EXECUTE FORMAT(
        'ALTER TABLE %I ALTER COLUMN master_key SET DEFAULT next_business_key(%L, %L)',
        p_source_table,
        p_source_table,
        p_key_prefix
    );
    EXECUTE FORMAT('ALTER TABLE %I ALTER COLUMN master_key SET NOT NULL', p_source_table);
    EXECUTE FORMAT('CREATE UNIQUE INDEX IF NOT EXISTS %I ON %I(master_key)', 'uq_' || p_source_table || '_master_key', p_source_table);
END;
$$;

DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT source_table, key_prefix FROM module_master_source_catalog ORDER BY source_table LOOP
        PERFORM ensure_source_master_key(rec.source_table, rec.key_prefix);
    END LOOP;
END;
$$;

CREATE OR REPLACE FUNCTION upsert_module_masters_for_source(p_source_table TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    rec RECORD;
    v_master_table TEXT;
    v_sql TEXT;
BEGIN
    SELECT *
    INTO rec
    FROM module_master_source_catalog
    WHERE source_table = p_source_table;

    IF rec.source_table IS NULL OR NOT module_table_exists(rec.source_table) THEN
        RETURN;
    END IF;

    v_master_table := rec.module_name || '_masters';

    EXECUTE FORMAT('DELETE FROM %I WHERE legacy_table = $1', v_master_table)
    USING rec.source_table;

    v_sql := FORMAT(
        'INSERT INTO %I (
            master_key, entity_type, legacy_table, legacy_pk, legacy_id,
            title, name, code, status,
            organization_id, department_id, project_id, requirement_id, workflow_id, task_id,
            actor_id, actor_type, core_data, metadata, created_at, updated_at
         )
         SELECT
            t.master_key,
            %L,
            %L,
            %s,
            t.legacy_id,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            %s,
            to_jsonb(t),
            %s,
            %s,
            %s
         FROM %I t
         ON CONFLICT (legacy_table, legacy_pk) DO UPDATE SET
            master_key = EXCLUDED.master_key,
            entity_type = EXCLUDED.entity_type,
            legacy_id = EXCLUDED.legacy_id,
            title = EXCLUDED.title,
            name = EXCLUDED.name,
            code = EXCLUDED.code,
            status = EXCLUDED.status,
            organization_id = EXCLUDED.organization_id,
            department_id = EXCLUDED.department_id,
            project_id = EXCLUDED.project_id,
            requirement_id = EXCLUDED.requirement_id,
            workflow_id = EXCLUDED.workflow_id,
            task_id = EXCLUDED.task_id,
            actor_id = EXCLUDED.actor_id,
            actor_type = EXCLUDED.actor_type,
            core_data = EXCLUDED.core_data,
            metadata = EXCLUDED.metadata,
            updated_at = EXCLUDED.updated_at',
        v_master_table,
        rec.entity_type,
        rec.source_table,
        module_legacy_pk_expr(rec.source_table, 't'),
        module_first_text_expr(rec.source_table, ARRAY['title', 'name', 'display_name', 'email', 'username', 'code', 'key', 'action', 'resource', 'file_name'], 't'),
        module_first_text_expr(rec.source_table, ARRAY['name', 'title', 'display_name', 'email', 'username', 'code', 'key', 'file_name'], 't'),
        module_first_text_expr(rec.source_table, ARRAY['code', 'key', 'slug', 'name', 'email'], 't'),
        module_first_text_expr(rec.source_table, ARRAY['status', 'state', 'decision', 'result'], 't'),
        module_uuid_expr(rec.source_table, 'organization_id', 't'),
        module_uuid_expr(rec.source_table, 'department_id', 't'),
        module_uuid_expr(rec.source_table, 'project_id', 't'),
        module_uuid_expr(rec.source_table, 'requirement_id', 't'),
        module_uuid_expr(rec.source_table, 'workflow_id', 't'),
        module_uuid_expr(rec.source_table, 'task_id', 't'),
        module_uuid_expr(rec.source_table, 'actor_id', 't'),
        module_first_text_expr(rec.source_table, ARRAY['actor_type', 'created_by_type', 'uploaded_by_type', 'submitted_by_type', 'evaluator_type', 'member_type'], 't'),
        module_jsonb_expr(rec.source_table, 'metadata', 't'),
        module_timestamp_expr(rec.source_table, 'created_at', 't'),
        module_timestamp_expr(rec.source_table, 'updated_at', 't'),
        rec.source_table
    );

    EXECUTE v_sql;
END;
$$;

CREATE OR REPLACE FUNCTION upsert_module_details_for_source(p_source_table TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    rec RECORD;
    v_detail_table TEXT;
    v_master_table TEXT;
    v_sql TEXT;
BEGIN
    SELECT *
    INTO rec
    FROM module_master_source_catalog
    WHERE source_table = p_source_table;

    IF rec.source_table IS NULL
       OR rec.relation_mode <> 'detail'
       OR rec.parent_table IS NULL
       OR rec.parent_fk IS NULL
       OR NOT module_table_exists(rec.source_table)
       OR NOT module_table_exists(rec.parent_table)
       OR NOT module_column_exists(rec.source_table, rec.parent_fk)
    THEN
        RETURN;
    END IF;

    v_detail_table := rec.module_name || '_details';
    v_master_table := rec.module_name || '_masters';

    EXECUTE FORMAT('DELETE FROM %I WHERE legacy_table = $1', v_detail_table)
    USING rec.source_table;

    v_sql := FORMAT(
        'INSERT INTO %I (
            sub_key, master_key, source_master_key, detail_type, legacy_table, legacy_pk, legacy_id,
            parent_legacy_table, parent_legacy_pk, parent_legacy_id, line_no,
            payload, metadata, created_at, updated_at
         )
         SELECT
            child.master_key,
            COALESCE(parent_master.master_key, child_master.master_key),
            child.master_key,
            %L,
            %L,
            %s,
            child.legacy_id,
            %L,
            COALESCE(%s, ''''),
            parent_old.legacy_id,
            ROW_NUMBER() OVER (PARTITION BY COALESCE(parent_master.master_key, child_master.master_key) ORDER BY %s),
            to_jsonb(child),
            %s,
            %s,
            %s
         FROM %I child
         JOIN %I child_master ON child_master.legacy_table = %L AND child_master.legacy_pk = %s
         LEFT JOIN %I parent_old ON child.%I::TEXT = %s
         LEFT JOIN %I parent_master ON parent_master.legacy_table = %L AND parent_master.legacy_pk = %s
         ON CONFLICT (legacy_table, legacy_pk) DO UPDATE SET
            master_key = EXCLUDED.master_key,
            source_master_key = EXCLUDED.source_master_key,
            detail_type = EXCLUDED.detail_type,
            legacy_id = EXCLUDED.legacy_id,
            parent_legacy_table = EXCLUDED.parent_legacy_table,
            parent_legacy_pk = EXCLUDED.parent_legacy_pk,
            parent_legacy_id = EXCLUDED.parent_legacy_id,
            line_no = EXCLUDED.line_no,
            payload = EXCLUDED.payload,
            metadata = EXCLUDED.metadata,
            updated_at = EXCLUDED.updated_at',
        v_detail_table,
        rec.entity_type,
        rec.source_table,
        module_legacy_pk_expr(rec.source_table, 'child'),
        rec.parent_table,
        module_legacy_pk_expr(rec.parent_table, 'parent_old'),
        module_legacy_pk_expr(rec.source_table, 'child'),
        module_jsonb_expr(rec.source_table, 'metadata', 'child'),
        module_timestamp_expr(rec.source_table, 'created_at', 'child'),
        module_timestamp_expr(rec.source_table, 'updated_at', 'child'),
        rec.source_table,
        v_master_table,
        rec.source_table,
        module_legacy_pk_expr(rec.source_table, 'child'),
        rec.parent_table,
        rec.parent_fk,
        module_legacy_pk_expr(rec.parent_table, 'parent_old'),
        v_master_table,
        rec.parent_table,
        module_legacy_pk_expr(rec.parent_table, 'parent_old')
    );

    EXECUTE v_sql;
END;
$$;

CREATE OR REPLACE FUNCTION refresh_module_source(p_source_table TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    child_rec RECORD;
BEGIN
    PERFORM upsert_module_masters_for_source(p_source_table);
    PERFORM upsert_module_details_for_source(p_source_table);

    FOR child_rec IN
        SELECT source_table
        FROM module_master_source_catalog
        WHERE parent_table = p_source_table
          AND module_table_exists(source_table)
        ORDER BY source_table
    LOOP
        PERFORM upsert_module_details_for_source(child_rec.source_table);
    END LOOP;
END;
$$;

CREATE OR REPLACE FUNCTION refresh_all_module_master_detail()
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT source_table
        FROM module_master_source_catalog
        WHERE module_table_exists(source_table)
        ORDER BY relation_mode, source_table
    LOOP
        PERFORM refresh_module_source(rec.source_table);
    END LOOP;
END;
$$;

SELECT refresh_all_module_master_detail();

CREATE OR REPLACE FUNCTION refresh_module_source_trigger()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM refresh_module_source(TG_TABLE_NAME);
    RETURN NULL;
END;
$$;

DO $$
DECLARE
    rec RECORD;
    v_trigger_name TEXT;
BEGIN
    FOR rec IN
        SELECT source_table
        FROM module_master_source_catalog
        WHERE module_table_exists(source_table)
        ORDER BY source_table
    LOOP
        v_trigger_name := 'trg_refresh_' || rec.source_table || '_module_master';
        EXECUTE FORMAT('DROP TRIGGER IF EXISTS %I ON %I', v_trigger_name, rec.source_table);
        EXECUTE FORMAT(
            'CREATE TRIGGER %I
             AFTER INSERT OR UPDATE OR DELETE ON %I
             FOR EACH STATEMENT EXECUTE FUNCTION refresh_module_source_trigger()',
            v_trigger_name,
            rec.source_table
        );
    END LOOP;
END;
$$;

CREATE TABLE IF NOT EXISTS module_master_migration_audit (
    module_name            TEXT NOT NULL,
    source_table           TEXT NOT NULL PRIMARY KEY,
    entity_type            TEXT NOT NULL,
    source_count           BIGINT NOT NULL DEFAULT 0,
    master_count           BIGINT NOT NULL DEFAULT 0,
    detail_count           BIGINT NOT NULL DEFAULT 0,
    status                 TEXT NOT NULL DEFAULT 'pending',
    checked_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION refresh_module_master_migration_audit()
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    rec RECORD;
    v_source_count BIGINT;
    v_master_count BIGINT;
    v_detail_count BIGINT;
BEGIN
    FOR rec IN
        SELECT *
        FROM module_master_source_catalog
        WHERE module_table_exists(source_table)
        ORDER BY source_table
    LOOP
        EXECUTE FORMAT('SELECT COUNT(*) FROM %I', rec.source_table) INTO v_source_count;
        EXECUTE FORMAT('SELECT COUNT(*) FROM %I WHERE legacy_table = $1', rec.module_name || '_masters')
            INTO v_master_count
            USING rec.source_table;

        IF rec.relation_mode = 'detail' THEN
            EXECUTE FORMAT('SELECT COUNT(*) FROM %I WHERE legacy_table = $1', rec.module_name || '_details')
                INTO v_detail_count
                USING rec.source_table;
        ELSE
            v_detail_count := 0;
        END IF;

        INSERT INTO module_master_migration_audit(
            module_name, source_table, entity_type, source_count, master_count, detail_count, status, checked_at
        )
        VALUES (
            rec.module_name,
            rec.source_table,
            rec.entity_type,
            v_source_count,
            v_master_count,
            v_detail_count,
            CASE
                WHEN v_source_count = v_master_count
                 AND (rec.relation_mode = 'master' OR v_source_count = v_detail_count)
                THEN 'ok'
                ELSE 'mismatch'
            END,
            NOW()
        )
        ON CONFLICT (source_table) DO UPDATE SET
            module_name = EXCLUDED.module_name,
            entity_type = EXCLUDED.entity_type,
            source_count = EXCLUDED.source_count,
            master_count = EXCLUDED.master_count,
            detail_count = EXCLUDED.detail_count,
            status = EXCLUDED.status,
            checked_at = EXCLUDED.checked_at;
    END LOOP;
END;
$$;

SELECT refresh_module_master_migration_audit();

CREATE OR REPLACE FUNCTION resolve_legacy_uuid(p_source_table TEXT, p_key TEXT)
RETURNS UUID
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    v_id UUID;
BEGIN
    IF p_key IS NULL OR p_key = '' THEN
        RETURN NULL;
    END IF;

    IF p_key ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN
        RETURN p_key::UUID;
    END IF;

    IF NOT module_table_exists(p_source_table) OR NOT module_column_exists(p_source_table, 'master_key') THEN
        RETURN NULL;
    END IF;

    EXECUTE FORMAT('SELECT legacy_id FROM %I WHERE master_key = $1 LIMIT 1', p_source_table)
    INTO v_id
    USING p_key;

    RETURN v_id;
END;
$$;

INSERT INTO data_field_catalog(table_name, field_name, data_type, display_name, is_master_key, is_sub_key, is_visible_default, display_order, metadata)
SELECT
    c.table_name,
    c.column_name,
    c.data_type,
    c.column_name,
    c.column_name = 'master_key',
    c.column_name = 'sub_key',
    c.column_name NOT IN ('core_data', 'payload', 'metadata'),
    c.ordinal_position,
    jsonb_build_object('canonical', true)
FROM information_schema.columns c
JOIN data_table_catalog t ON t.table_name = c.table_name
WHERE c.table_schema = 'public'
  AND t.category = 'canonical'
ON CONFLICT (table_name, field_name) DO UPDATE SET
    data_type = EXCLUDED.data_type,
    display_name = EXCLUDED.display_name,
    is_master_key = EXCLUDED.is_master_key,
    is_sub_key = EXCLUDED.is_sub_key,
    is_visible_default = EXCLUDED.is_visible_default,
    display_order = EXCLUDED.display_order,
    metadata = data_field_catalog.metadata || EXCLUDED.metadata,
    updated_at = NOW();
