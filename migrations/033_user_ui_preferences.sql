CREATE TABLE IF NOT EXISTS user_ui_preferences (
    actor_id       TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    value          JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (actor_id, preference_key)
);

CREATE INDEX IF NOT EXISTS idx_user_ui_preferences_key
    ON user_ui_preferences(preference_key);
