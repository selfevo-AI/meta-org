-- Backfill fixes for environments that applied an earlier assistant business interaction migration.

ALTER TABLE project_evaluations
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

WITH agent_defaults AS (
    SELECT id AS agent_id
    FROM ai_agents
    WHERE is_active
    ORDER BY updated_at DESC, created_at DESC
    LIMIT 1
)
UPDATE assistant_module_defaults
SET agent_id = agent_defaults.agent_id,
    updated_at = NOW()
FROM agent_defaults
WHERE assistant_module_defaults.agent_id IS NULL;

INSERT INTO assistant_module_defaults (module_key, target_type, agent_id, provider_id, provider_type, model, metadata)
SELECT module_key, target_type, agent_id, provider_id, provider_type, model_key,
    jsonb_build_object('source', 'patch_seed_first_active_model')
FROM (
    SELECT *
    FROM (VALUES
        ('meta_org', ''),
        ('project_cost', 'project_cost')
    ) AS defaults(module_key, target_type)
) defaults
CROSS JOIN LATERAL (
    SELECT m.provider_id, mp.provider_type, m.model_key
    FROM models m
    JOIN model_providers mp ON mp.id = m.provider_id
    WHERE m.status = 'active' AND mp.status = 'active'
    ORDER BY m.updated_at DESC, m.created_at DESC
    LIMIT 1
) model_defaults
LEFT JOIN LATERAL (
    SELECT id AS agent_id
    FROM ai_agents
    WHERE is_active
    ORDER BY updated_at DESC, created_at DESC
    LIMIT 1
) agent_defaults ON TRUE
ON CONFLICT (module_key, target_type) DO UPDATE
SET agent_id = COALESCE(assistant_module_defaults.agent_id, EXCLUDED.agent_id),
    provider_id = COALESCE(assistant_module_defaults.provider_id, EXCLUDED.provider_id),
    provider_type = COALESCE(NULLIF(assistant_module_defaults.provider_type, ''), EXCLUDED.provider_type),
    model = COALESCE(NULLIF(assistant_module_defaults.model, ''), EXCLUDED.model),
    updated_at = NOW();
