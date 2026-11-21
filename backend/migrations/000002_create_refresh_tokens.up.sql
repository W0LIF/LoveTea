CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    token_hash BYTEA NOT NULL UNIQUE,

    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens(id),

    CONSTRAINT refresh_tokens_expiration_check
        CHECK (expires_at > created_at)
);

CREATE INDEX refresh_tokens_user_id_idx
    ON refresh_tokens(user_id);

CREATE INDEX refresh_tokens_active_idx
    ON refresh_tokens(user_id, expires_at)
    WHERE revoked_at IS NULL;
