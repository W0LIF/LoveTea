CREATE TABLE couple_pet_state (
    couple_id UUID PRIMARY KEY
        REFERENCES couples(id) ON DELETE CASCADE,
    appearance JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT couple_pet_state_appearance_check
        CHECK (jsonb_typeof(appearance) = 'object'),
    CONSTRAINT couple_pet_state_version_check
        CHECK (version > 0)
);

INSERT INTO couple_pet_state (couple_id)
SELECT id
FROM couples
WHERE status IN ('pending', 'active')
ON CONFLICT (couple_id) DO NOTHING;
