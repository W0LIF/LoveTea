CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    email VARCHAR(254) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,

    email_verified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_email_not_empty
        CHECK (length(trim(email)) > 0),

    CONSTRAINT users_password_hash_not_empty
        CHECK (length(password_hash) > 0)
);

-- Email должен быть уникальным без учёта регистра.
CREATE UNIQUE INDEX users_email_lower_unique_idx
    ON users (LOWER(email));

-- Индекс для поиска пользователей по дате создания.
CREATE INDEX users_created_at_idx
    ON users (created_at DESC);
