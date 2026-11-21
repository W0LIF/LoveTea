CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY
        REFERENCES users(id) ON DELETE CASCADE,

    display_name VARCHAR(50),
    avatar_key TEXT,

    locale VARCHAR(10) NOT NULL DEFAULT 'ru',
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_profiles_display_name_length
        CHECK (
            display_name IS NULL
            OR char_length(trim(display_name)) BETWEEN 1 AND 50
        ),

    CONSTRAINT user_profiles_locale_check
        CHECK (locale IN ('ru', 'en'))
);

CREATE INDEX user_profiles_updated_at_idx
    ON user_profiles(updated_at);
