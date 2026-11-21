CREATE TABLE sync_cursors (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    cursor BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT sync_cursors_value_check
        CHECK (cursor >= 0)
);

CREATE TABLE sync_entity_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID NOT NULL,
    version BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT sync_entity_versions_scope_check
        CHECK ((owner_user_id IS NOT NULL) <> (couple_id IS NOT NULL)),
    CONSTRAINT sync_entity_versions_type_check
        CHECK (entity_type ~ '^[a-z][a-z0-9_]{0,79}$'),
    CONSTRAINT sync_entity_versions_version_check
        CHECK (version > 0)
);

CREATE UNIQUE INDEX sync_entity_versions_owner_unique_idx
    ON sync_entity_versions(owner_user_id, entity_type, entity_id)
    WHERE owner_user_id IS NOT NULL;

CREATE UNIQUE INDEX sync_entity_versions_couple_unique_idx
    ON sync_entity_versions(couple_id, entity_type, entity_id)
    WHERE couple_id IS NOT NULL;

CREATE TABLE sync_changes (
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_cursor BIGINT NOT NULL,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID NOT NULL,
    version BIGINT NOT NULL,
    operation VARCHAR(16) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT sync_changes_cursor_check
        CHECK (user_cursor > 0),
    CONSTRAINT sync_changes_type_check
        CHECK (entity_type ~ '^[a-z][a-z0-9_]{0,79}$'),
    CONSTRAINT sync_changes_version_check
        CHECK (version > 0),
    CONSTRAINT sync_changes_operation_check
        CHECK (operation IN ('upsert', 'delete')),
    CONSTRAINT sync_changes_payload_check
        CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT sync_changes_recipient_cursor_unique
        UNIQUE (recipient_user_id, user_cursor)
);

CREATE INDEX sync_changes_user_cursor_idx
    ON sync_changes(recipient_user_id, user_cursor);

CREATE INDEX sync_changes_couple_cursor_idx
    ON sync_changes(recipient_user_id, couple_id, user_cursor)
    WHERE couple_id IS NOT NULL;
