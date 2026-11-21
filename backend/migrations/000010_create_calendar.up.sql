CREATE TABLE calendar_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_type VARCHAR(20) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    color VARCHAR(7),
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT calendar_events_title_check
        CHECK (char_length(trim(title)) BETWEEN 1 AND 200),
    CONSTRAINT calendar_events_description_length_check
        CHECK (char_length(description) <= 5000),
    CONSTRAINT calendar_events_type_check
        CHECK (event_type IN ('free', 'busy', 'date', 'meeting', 'task')),
    CONSTRAINT calendar_events_time_check
        CHECK (ends_at > starts_at),
    CONSTRAINT calendar_events_color_check
        CHECK (color IS NULL OR color ~ '^#[0-9A-Fa-f]{6}$'),
    CONSTRAINT calendar_events_version_check
        CHECK (version > 0)
);

CREATE INDEX calendar_events_personal_start_idx
    ON calendar_events(owner_user_id, starts_at, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL;

CREATE INDEX calendar_events_couple_start_idx
    ON calendar_events(couple_id, starts_at, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE important_dates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    event_date DATE NOT NULL,
    repeats_yearly BOOLEAN NOT NULL DEFAULT FALSE,
    reminder_days SMALLINT[] NOT NULL DEFAULT '{}',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT important_dates_title_check
        CHECK (char_length(trim(title)) BETWEEN 1 AND 200),
    CONSTRAINT important_dates_reminder_days_check
        CHECK (
            cardinality(reminder_days) <= 10
            AND 0 <= ALL(reminder_days)
            AND 365 >= ALL(reminder_days)
        ),
    CONSTRAINT important_dates_version_check
        CHECK (version > 0)
);

CREATE INDEX important_dates_personal_date_idx
    ON important_dates(owner_user_id, event_date, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL;

CREATE INDEX important_dates_couple_date_idx
    ON important_dates(couple_id, event_date, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL;
