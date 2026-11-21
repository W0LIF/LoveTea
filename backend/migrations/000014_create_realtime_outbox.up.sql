CREATE TABLE realtime_outbox (
    id BIGSERIAL PRIMARY KEY,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sync_cursor BIGINT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT realtime_outbox_attempts_check CHECK (attempts >= 0),
    CONSTRAINT realtime_outbox_change_fk
        FOREIGN KEY (recipient_user_id, sync_cursor)
        REFERENCES sync_changes(recipient_user_id, user_cursor)
        ON DELETE CASCADE,
    CONSTRAINT realtime_outbox_recipient_cursor_unique
        UNIQUE (recipient_user_id, sync_cursor)
);

CREATE INDEX realtime_outbox_pending_idx
    ON realtime_outbox(locked_until, id);
