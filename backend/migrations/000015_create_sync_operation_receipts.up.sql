CREATE TABLE sync_operation_receipts (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation_id UUID NOT NULL,
    request_hash BYTEA NOT NULL,
    result JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',

    PRIMARY KEY (user_id, operation_id),
    CONSTRAINT sync_operation_receipts_hash_check
        CHECK (octet_length(request_hash) = 32),
    CONSTRAINT sync_operation_receipts_result_check
        CHECK (result IS NULL OR jsonb_typeof(result) = 'object'),
    CONSTRAINT sync_operation_receipts_expiration_check
        CHECK (expires_at > created_at)
);

CREATE INDEX sync_operation_receipts_expiration_idx
    ON sync_operation_receipts(expires_at);
