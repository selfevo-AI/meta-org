CREATE TABLE IF NOT EXISTS model_provider_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES model_providers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    base_url TEXT NOT NULL DEFAULT '',
    encrypted_api_key TEXT NOT NULL,
    masked_api_key TEXT NOT NULL DEFAULT '',
    owner_type TEXT NOT NULL DEFAULT ''
        CHECK (owner_type IN ('', 'human', 'agent', 'team', 'system')),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES ai_agents(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled', 'error', 'quota_exhausted')),
    priority INT NOT NULL DEFAULT 50,
    concurrency_limit INT NOT NULL DEFAULT 0,
    inflight_requests INT NOT NULL DEFAULT 0,
    load_factor INT NOT NULL DEFAULT 1,
    rate_multiplier NUMERIC(12,6) NOT NULL DEFAULT 1,
    quota_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    quota_used NUMERIC(18,8) NOT NULL DEFAULT 0,
    quota_currency TEXT NOT NULL DEFAULT 'CNY',
    supported_model_patterns JSONB NOT NULL DEFAULT '[]',
    model_mapping JSONB NOT NULL DEFAULT '{}',
    health_status TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    last_tested_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_provider_channels_provider_status
    ON model_provider_channels(provider_id, status, priority);
CREATE INDEX IF NOT EXISTS idx_model_provider_channels_owner
    ON model_provider_channels(owner_type, user_id, agent_id);

ALTER TABLE model_price_versions
    ADD COLUMN IF NOT EXISTS cache_creation_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_read_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_5m_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_1h_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS image_output_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS priority_input_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS priority_output_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS priority_cache_read_price_per_1k NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS long_context_threshold INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS long_context_input_multiplier NUMERIC(12,6) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS long_context_output_multiplier NUMERIC(12,6) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS billing_mode TEXT NOT NULL DEFAULT 'token',
    ADD COLUMN IF NOT EXISTS pricing_source TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

ALTER TABLE ai_invocations
    ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES model_provider_channels(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS requested_model TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS upstream_model TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS model_mapping_chain TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS service_tier TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reasoning_effort TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cache_creation_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_read_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_5m_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_1h_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS image_output_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cost_breakdown JSONB NOT NULL DEFAULT '{}';

ALTER TABLE ai_usage_ledger
    ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES model_provider_channels(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cache_creation_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_read_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_5m_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_1h_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS image_output_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS input_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS output_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_read_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS image_output_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rate_multiplier NUMERIC(12,6) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS actual_amount NUMERIC(18,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS service_tier TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reasoning_effort TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS requested_model TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS upstream_model TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_ai_invocations_channel
    ON ai_invocations(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_usage_ledger_channel
    ON ai_usage_ledger(channel_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    provider_id UUID REFERENCES model_providers(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES model_provider_channels(id) ON DELETE CASCADE,
    match_scope TEXT NOT NULL DEFAULT 'global',
    match_value TEXT NOT NULL DEFAULT '',
    model_pattern TEXT NOT NULL DEFAULT '',
    priority INT NOT NULL DEFAULT 100,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_routing_rules_status_priority
    ON ai_routing_rules(status, priority);
