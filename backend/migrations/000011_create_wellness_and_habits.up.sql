CREATE TABLE mood_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    mood VARCHAR(32) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    shared_couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT mood_entries_mood_check
        CHECK (char_length(trim(mood)) BETWEEN 1 AND 32),
    CONSTRAINT mood_entries_note_length_check
        CHECK (char_length(note) <= 2000),
    CONSTRAINT mood_entries_version_check
        CHECK (version > 0)
);

CREATE UNIQUE INDEX mood_entries_owner_date_unique_idx
    ON mood_entries(owner_user_id, entry_date)
    WHERE deleted_at IS NULL;

CREATE INDEX mood_entries_shared_date_idx
    ON mood_entries(shared_couple_id, entry_date, id)
    WHERE shared_couple_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE sleep_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    sleep_started_at TIMESTAMPTZ NOT NULL,
    woke_at TIMESTAMPTZ NOT NULL,
    quality SMALLINT NOT NULL,
    shared_couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT sleep_entries_quality_check
        CHECK (quality BETWEEN 1 AND 5),
    CONSTRAINT sleep_entries_time_check
        CHECK (
            woke_at > sleep_started_at
            AND woke_at <= sleep_started_at + INTERVAL '24 hours'
        ),
    CONSTRAINT sleep_entries_version_check
        CHECK (version > 0)
);

CREATE UNIQUE INDEX sleep_entries_owner_date_unique_idx
    ON sleep_entries(owner_user_id, entry_date)
    WHERE deleted_at IS NULL;

CREATE INDEX sleep_entries_shared_date_idx
    ON sleep_entries(shared_couple_id, entry_date, id)
    WHERE shared_couple_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE habits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    target_per_week SMALLINT NOT NULL DEFAULT 7,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT habits_title_check
        CHECK (char_length(trim(title)) BETWEEN 1 AND 200),
    CONSTRAINT habits_description_length_check
        CHECK (char_length(description) <= 2000),
    CONSTRAINT habits_target_per_week_check
        CHECK (target_per_week BETWEEN 1 AND 7),
    CONSTRAINT habits_version_check
        CHECK (version > 0)
);

CREATE INDEX habits_owner_active_idx
    ON habits(owner_user_id, created_at, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL;

CREATE INDEX habits_couple_active_idx
    ON habits(couple_id, created_at, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE habit_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    completed_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ

);

CREATE UNIQUE INDEX habit_completions_active_unique_idx
    ON habit_completions(habit_id, user_id, completed_on)
    WHERE deleted_at IS NULL;

CREATE INDEX habit_completions_stats_idx
    ON habit_completions(habit_id, completed_on, user_id)
    WHERE deleted_at IS NULL;
