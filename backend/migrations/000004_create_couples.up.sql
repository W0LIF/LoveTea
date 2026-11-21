CREATE TABLE couples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    created_by UUID NOT NULL REFERENCES users(id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    detached_at TIMESTAMPTZ,

    CONSTRAINT couples_status_check
        CHECK (status IN ('pending', 'active', 'detached')),

    CONSTRAINT couples_activation_check
        CHECK (
            (status = 'pending' AND activated_at IS NULL AND detached_at IS NULL)
            OR
            (status = 'active' AND activated_at IS NOT NULL AND detached_at IS NULL)
            OR
            (status = 'detached' AND detached_at IS NOT NULL)
        )
);

CREATE INDEX couples_status_idx
    ON couples(status);


CREATE TABLE couple_members (
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    nickname VARCHAR(50),
    mascot_type VARCHAR(20) NOT NULL DEFAULT 'kitten',

    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,

    PRIMARY KEY (couple_id, user_id),

    CONSTRAINT couple_members_nickname_check
        CHECK (
            nickname IS NULL
            OR char_length(trim(nickname)) BETWEEN 1 AND 50
        ),

    CONSTRAINT couple_members_mascot_check
        CHECK (mascot_type IN ('kitten', 'wolf'))
);

-- Пользователь может состоять только в одной активной или ожидающей паре.
CREATE UNIQUE INDEX couple_members_active_user_unique_idx
    ON couple_members(user_id)
    WHERE left_at IS NULL;

CREATE INDEX couple_members_couple_idx
    ON couple_members(couple_id);


CREATE TABLE couple_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),

    -- Хеши вместо открытых кодов и токенов.
    code_hash BYTEA NOT NULL,
    token_hash BYTEA NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    accepted_by UUID REFERENCES users(id),
    revoked_at TIMESTAMPTZ,

    code_attempts INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT couple_invitations_code_hash_unique
        UNIQUE (code_hash),

    CONSTRAINT couple_invitations_token_hash_unique
        UNIQUE (token_hash),

    CONSTRAINT couple_invitations_expiration_check
        CHECK (expires_at > created_at),

    CONSTRAINT couple_invitations_attempts_check
        CHECK (code_attempts >= 0),

    CONSTRAINT couple_invitations_acceptance_check
        CHECK (
            (accepted_at IS NULL AND accepted_by IS NULL)
            OR
            (accepted_at IS NOT NULL AND accepted_by IS NOT NULL)
        )
);

-- У пары может быть только одно неотозванное и ещё не принятое приглашение.
CREATE UNIQUE INDEX couple_invitations_pending_unique_idx
    ON couple_invitations(couple_id)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;

CREATE INDEX couple_invitations_expires_at_idx
    ON couple_invitations(expires_at);


CREATE TABLE couple_detach_confirmations (
    couple_id UUID NOT NULL,
    user_id UUID NOT NULL,
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (couple_id, user_id),

    CONSTRAINT couple_detach_member_fk
        FOREIGN KEY (couple_id, user_id)
        REFERENCES couple_members(couple_id, user_id)
        ON DELETE CASCADE
);

CREATE INDEX couple_detach_confirmations_couple_idx
    ON couple_detach_confirmations(couple_id);
