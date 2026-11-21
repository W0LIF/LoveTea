CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    due_at TIMESTAMPTZ,
    reminder_at TIMESTAMPTZ,
    color VARCHAR(7),
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT tasks_title_check
        CHECK (char_length(trim(title)) BETWEEN 1 AND 200),
    CONSTRAINT tasks_description_length_check
        CHECK (char_length(description) <= 5000),
    CONSTRAINT tasks_status_check
        CHECK (status IN ('open', 'in_progress', 'done', 'cancelled')),
    CONSTRAINT tasks_reminder_check
        CHECK (reminder_at IS NULL OR (due_at IS NOT NULL AND reminder_at <= due_at)),
    CONSTRAINT tasks_version_check
        CHECK (version > 0),
    CONSTRAINT tasks_color_check
        CHECK (color IS NULL OR color ~ '^#[0-9A-Fa-f]{6}$')
);

CREATE INDEX tasks_personal_due_idx
    ON tasks(owner_user_id, due_at, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL;

CREATE INDEX tasks_couple_due_idx
    ON tasks(couple_id, due_at, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL;
