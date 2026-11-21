-- Миграция 000017 — все новые модули: chat, letters, notifications, games, wheel,
-- location, date ideas, memories E2E, account export, recurring events, reminders.
--
-- Каждая таблица привязана к scope: owner_user_id (личные данные) или
-- couple_id (общие данные).

-- ============================================================
-- CHAT (сообщения пары, реакции)
-- ============================================================
CREATE TABLE chat_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chat_conversation_couple_unique
        UNIQUE (couple_id)
);

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    -- E2E: null = plaintext, not null = encrypted payload
    ciphertext TEXT,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chat_message_content_check
        CHECK (char_length(content) BETWEEN 1 AND 5000),
    CONSTRAINT chat_message_version_check
        CHECK (version > 0)
);

CREATE INDEX chat_messages_conversation_created_idx
    ON chat_messages(conversation_id, created_at DESC, id)
    WHERE deleted_at IS NULL;

CREATE TABLE chat_message_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chat_reaction_unique
        UNIQUE (message_id, user_id, emoji)
);

-- ============================================================
-- LETTERS — «Гневное письмо» / «Спокойное письмо» (cooldown)
-- ============================================================
CREATE TABLE letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,

    content TEXT NOT NULL,
    cooldown_until TIMESTAMPTZ NOT NULL,

    -- статус: 'draft' | 'waiting' | 'sent' | 'read' | 'archived'
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT letter_content_check
        CHECK (char_length(content) BETWEEN 1 AND 10000),
    CONSTRAINT letter_status_check
        CHECK (status IN ('draft', 'waiting', 'sent', 'read', 'archived')),
    CONSTRAINT letter_version_check
        CHECK (version > 0)
);

CREATE INDEX letter_sender_status_idx
    ON letters(sender_user_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX letter_recipient_status_idx
    ON letters(recipient_user_id, status)
    WHERE deleted_at IS NULL;

-- ============================================================
-- NOTIFICATIONS — настройки и журнал
-- ============================================================
CREATE TABLE notification_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- какие категории уведомлений включены
    enabled_categories JSONB NOT NULL DEFAULT '["mood", "task", "event", "date", "habit", "message", "achievement"]'::jsonb,

    -- тихие часы (не присылать push)
    quiet_hours_start INTEGER NOT NULL DEFAULT 22,
    quiet_hours_end INTEGER NOT NULL DEFAULT 7,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notification_settings_categories_check
        CHECK (jsonb_typeof(enabled_categories) = 'array')
);

CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    body TEXT NOT NULL,
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notification_category_check
        CHECK (char_length(category) > 0),
    CONSTRAINT notification_title_check
        CHECK (char_length(title) > 0)
);

CREATE INDEX notification_user_unread_idx
    ON notifications(user_id, created_at DESC)
    WHERE is_read = false;

-- ============================================================
-- GAMES — «Насколько хорошо ты меня знаешь» + мини-игры
-- ============================================================
CREATE TABLE game_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    player_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    opponent_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- type: 'quiz' | 'mini'
    type VARCHAR(20) NOT NULL DEFAULT 'quiz',
    total_questions SMALLINT NOT NULL DEFAULT 10,
    correct_answers SMALLINT NOT NULL DEFAULT 0,
    score SMALLINT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'in_progress',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,

    CONSTRAINT game_session_type_check
        CHECK (type IN ('quiz', 'mini')),
    CONSTRAINT game_session_score_check
        CHECK (score >= 0),
    CONSTRAINT game_session_status_check
        CHECK (status IN ('in_progress', 'finished', 'cancelled'))
);

CREATE INDEX game_session_couple_player_idx
    ON game_sessions(couple_id, player_user_id, started_at DESC);

CREATE TABLE achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- null = global, couple_id = couple-specific
    couple_id UUID REFERENCES couples(id) ON DELETE CASCADE,

    key VARCHAR(80) NOT NULL,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon VARCHAR(50) NOT NULL DEFAULT 'trophy',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT achievement_key_unique
        UNIQUE (key, couple_id),
    CONSTRAINT achievement_title_check
        CHECK (char_length(title) > 0)
);

CREATE TABLE achievement_unlocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    achievement_id UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT achievement_unlock_unique
        UNIQUE (achievement_id, user_id)
);

-- ============================================================
-- WHEEL — колесо с правилами согласия
-- ============================================================
CREATE TABLE wheel_sectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    requires_confirmation BOOLEAN NOT NULL DEFAULT false,
    position SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT wheel_sector_title_check
        CHECK (char_length(title) > 0),
    CONSTRAINT wheel_sector_position_check
        CHECK (position >= 0)
);

CREATE INDEX wheel_sector_couple_pos_idx
    ON wheel_sectors(couple_id, position)
    WHERE deleted_at IS NULL;

CREATE TABLE wheel_spins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    spun_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    selected_sector_id UUID,
    confirmed_by_both BOOLEAN NOT NULL DEFAULT false,
    cancelled BOOLEAN NOT NULL DEFAULT false,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT wheel_spin_sector_fk
        FOREIGN KEY (couple_id, selected_sector_id)
        REFERENCES wheel_sectors(couple_id, id)
        ON DELETE SET NULL
);

CREATE INDEX wheel_spin_couple_created_idx
    ON wheel_spins(couple_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- ============================================================
-- LOCATION — геолокация с добровольным согласием
-- ============================================================
CREATE TABLE location_sharing (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT false,
    last_sharing_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT location_sharing_couple_unique
        UNIQUE (couple_id, user_id)
);

CREATE TABLE location_positions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    accuracy DOUBLE PRECISION,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Удаление старых позиций через retention period
    CONSTRAINT location_position_accuracy_check
        CHECK (accuracy IS NULL OR accuracy > 0)
);

CREATE INDEX location_position_user_couple_idx
    ON location_positions(user_id, couple_id, captured_at DESC);

CREATE TABLE location_geofences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    radius_meters DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT geofence_name_check
        CHECK (char_length(name) > 0),
    CONSTRAINT geofence_radius_check
        CHECK (radius_meters > 0 AND radius_meters <= 100000)
);

CREATE INDEX geofence_couple_user_idx
    ON location_geofences(couple_id, user_id)
    WHERE deleted_at IS NULL;

-- ============================================================
-- DATE IDEAS — идеи свиданий + рекомендации
-- ============================================================
CREATE TABLE date_ideas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    proposer_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    budget_min INTEGER,
    budget_max INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'suggested',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT date_idea_title_check
        CHECK (char_length(title) > 0),
    CONSTRAINT date_idea_category_check
        CHECK (category IN ('indoor', 'outdoor', 'active', 'cozy', 'creative', 'general')),
    CONSTRAINT date_idea_status_check
        CHECK (status IN ('suggested', 'accepted', 'planned', 'completed', 'cancelled', 'rejected'))
);

CREATE INDEX date_idea_couple_status_idx
    ON date_ideas(couple_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE date_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    suggested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT rec_title_check
        CHECK (char_length(title) > 0)
);

CREATE INDEX rec_user_couple_idx
    ON date_recommendations(user_id, couple_id, suggested_at DESC);

-- ============================================================
-- MEMORIES E2E — блокировка снимается, хранится ciphertext
-- ============================================================
-- Таблица memories уже существует (000012). Добавляем E2E-поля.
ALTER TABLE memories ADD COLUMN IF NOT EXISTS content_encrypted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS ciphertext TEXT;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS cipher_nonce TEXT;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS cipher_key_ref TEXT;

-- ============================================================
-- ACCOUNT EXPORT & DELETE
-- ============================================================
CREATE TABLE account_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    file_url TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CONSTRAINT export_status_check
        CHECK (status IN ('pending', 'ready', 'expired'))
);

-- ============================================================
-- RECURRING EVENTS + REMINDERS
-- ============================================================
ALTER TABLE calendar_events ADD COLUMN IF NOT EXISTS recurrence_rule TEXT;

-- reminder_events — расширенные напоминания
CREATE TABLE reminder_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    reminder_minutes_before INTEGER NOT NULL DEFAULT 30,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT reminder_minutes_check
        CHECK (reminder_minutes_before BETWEEN -1440 AND 1440)
);

-- reminder_log — журнал отправленных уведомлений
CREATE TABLE reminder_log (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    channel VARCHAR(20) NOT NULL DEFAULT 'push',

    CONSTRAINT reminder_log_channel_check
        CHECK (channel IN ('push', 'in_app', 'email'))
);

CREATE INDEX reminder_log_user_sent_idx
    ON reminder_log(user_id, sent_at DESC);
