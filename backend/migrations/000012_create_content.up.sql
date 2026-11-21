CREATE TABLE media_files (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL UNIQUE,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT media_files_content_type_check
        CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    CONSTRAINT media_files_size_check
        CHECK (size_bytes IS NULL OR size_bytes BETWEEN 1 AND 20971520),
    CONSTRAINT media_files_status_check
        CHECK (status IN ('pending', 'ready', 'deleted')),
    CONSTRAINT media_files_version_check
        CHECK (version > 0),
    CONSTRAINT media_files_ready_size_check
        CHECK (status <> 'ready' OR size_bytes IS NOT NULL)
);

CREATE INDEX media_files_owner_active_idx
    ON media_files(owner_user_id, created_at DESC, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL AND status = 'ready';

CREATE INDEX media_files_couple_active_idx
    ON media_files(couple_id, created_at DESC, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL AND status = 'ready';

CREATE TABLE moments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    caption TEXT NOT NULL DEFAULT '',
    happened_at TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT moments_title_check
        CHECK (char_length(trim(title)) BETWEEN 1 AND 200),
    CONSTRAINT moments_caption_length_check
        CHECK (char_length(caption) <= 5000),
    CONSTRAINT moments_version_check
        CHECK (version > 0)
);

CREATE INDEX moments_owner_happened_idx
    ON moments(owner_user_id, happened_at DESC, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL;

CREATE INDEX moments_couple_happened_idx
    ON moments(couple_id, happened_at DESC, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE moment_media (
    moment_id UUID NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media_files(id) ON DELETE RESTRICT,
    position SMALLINT NOT NULL,

    PRIMARY KEY (moment_id, media_id),
    CONSTRAINT moment_media_position_check CHECK (position BETWEEN 0 AND 19)
);

CREATE TABLE memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT memories_title_check
        CHECK (char_length(trim(title)) BETWEEN 1 AND 200),
    CONSTRAINT memories_content_length_check
        CHECK (char_length(content) BETWEEN 1 AND 10000),
    CONSTRAINT memories_version_check
        CHECK (version > 0)
);

CREATE INDEX memories_owner_created_idx
    ON memories(owner_user_id, created_at DESC, id)
    WHERE couple_id IS NULL AND deleted_at IS NULL;

CREATE INDEX memories_couple_created_idx
    ON memories(couple_id, created_at DESC, id)
    WHERE couple_id IS NOT NULL AND deleted_at IS NULL;
