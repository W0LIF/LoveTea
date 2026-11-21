ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,

    CONSTRAINT password_reset_tokens_expiration_check
        CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX password_reset_tokens_active_user_unique_idx
    ON password_reset_tokens(user_id)
    WHERE used_at IS NULL;

CREATE INDEX password_reset_tokens_expiration_idx
    ON password_reset_tokens(expires_at);
