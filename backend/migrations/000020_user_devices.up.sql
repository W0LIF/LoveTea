-- user_devices — 등록된 устройства для push-уведомлений (FCM / APNs)
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    push_token TEXT NOT NULL,
    app_version VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_device_platform_check
        CHECK (platform IN ('android', 'ios')),
    CONSTRAINT user_device_push_token_not_empty
        CHECK (length(trim(push_token)) > 0),
    CONSTRAINT user_device_unique
        UNIQUE (user_id, device_id, platform)
);

CREATE INDEX user_device_user_platform_idx
    ON user_devices(user_id, platform);

-- Down
DROP TABLE IF EXISTS user_devices;
