-- 010_evolution.sql

CREATE TABLE weight_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id              UUID NOT NULL,
    actor_type            TEXT NOT NULL,
    overall_score         DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    expertise_score       DOUBLE PRECISION DEFAULT 0,
    track_record_score    DOUBLE PRECISION DEFAULT 0,
    reliability_score     DOUBLE PRECISION DEFAULT 0,
    recency_score         DOUBLE PRECISION DEFAULT 0,
    context_fit_score     DOUBLE PRECISION DEFAULT 0,
    principle_score       DOUBLE PRECISION DEFAULT 0,
    decision_count        INT NOT NULL DEFAULT 0,
    last_updated          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (actor_id, actor_type)
);

CREATE TABLE weight_alphas (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alpha_expertise     DOUBLE PRECISION NOT NULL DEFAULT 0.25,
    alpha_track_record  DOUBLE PRECISION NOT NULL DEFAULT 0.20,
    alpha_reliability   DOUBLE PRECISION NOT NULL DEFAULT 0.15,
    alpha_recency       DOUBLE PRECISION NOT NULL DEFAULT 0.10,
    alpha_context_fit   DOUBLE PRECISION NOT NULL DEFAULT 0.10,
    alpha_principle     DOUBLE PRECISION NOT NULL DEFAULT 0.20,
    version             INT NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE experiments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    hypothesis          TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'proposed',
    mvru_id             UUID,
    alpha_overrides     JSONB,
    success_criteria    JSONB NOT NULL DEFAULT '{}',
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    conclusion          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE knowledge_entries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id         UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    title               TEXT NOT NULL,
    content             TEXT NOT NULL,
    tags                TEXT[] DEFAULT '{}',
    source              TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE signals (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_type         TEXT NOT NULL,
    source              TEXT NOT NULL,
    priority            INT NOT NULL DEFAULT 0,
    data                JSONB NOT NULL DEFAULT '{}',
    acknowledged        BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_weight_actor ON weight_scores(actor_id, actor_type);
CREATE INDEX idx_weight_overall ON weight_scores(overall_score DESC);
CREATE INDEX idx_experiment_status ON experiments(status);
CREATE INDEX idx_knowledge_tags ON knowledge_entries USING GIN(tags);
CREATE INDEX idx_knowledge_source ON knowledge_entries(source);
CREATE INDEX idx_signals_priority ON signals(priority DESC, created_at DESC);
CREATE INDEX idx_signals_acknowledged ON signals(acknowledged);
