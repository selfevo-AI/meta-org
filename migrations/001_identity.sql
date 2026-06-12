-- 001_identity.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE role_type AS ENUM ('planner', 'executor', 'reviewer');
CREATE TYPE permission_level AS ENUM ('L1', 'L2', 'L3', 'L4');

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_agents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    model_type      VARCHAR(100) NOT NULL,
    api_key_hash    VARCHAR(255) NOT NULL,
    capabilities    JSONB NOT NULL DEFAULT '[]',
    permission_level permission_level NOT NULL DEFAULT 'L2',
    metadata        JSONB DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    role_type       role_type NOT NULL,
    description     TEXT,
    permissions     JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_roles (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE agent_roles (
    agent_id        UUID NOT NULL REFERENCES ai_agents(id) ON DELETE CASCADE,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, role_id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_agents_model ON ai_agents(model_type);
