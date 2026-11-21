CREATE TABLE onboarding_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(80) NOT NULL,
    version INTEGER NOT NULL,
    title JSONB NOT NULL,
    description JSONB NOT NULL DEFAULT '{}'::jsonb,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT onboarding_tests_slug_check
        CHECK (slug ~ '^[a-z0-9][a-z0-9_-]{0,79}$'),
    CONSTRAINT onboarding_tests_version_check
        CHECK (version > 0),
    CONSTRAINT onboarding_tests_title_check
        CHECK (jsonb_typeof(title) = 'object' AND title <> '{}'::jsonb),
    CONSTRAINT onboarding_tests_description_check
        CHECK (jsonb_typeof(description) = 'object'),
    CONSTRAINT onboarding_tests_slug_version_unique
        UNIQUE (slug, version)
);

CREATE INDEX onboarding_tests_published_idx
    ON onboarding_tests(slug, version DESC)
    WHERE published_at IS NOT NULL;

CREATE TABLE onboarding_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id UUID NOT NULL
        REFERENCES onboarding_tests(id) ON DELETE CASCADE,
    question_key VARCHAR(80) NOT NULL,
    prompt JSONB NOT NULL,
    answer_type VARCHAR(24) NOT NULL,
    position INTEGER NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT TRUE,
    options JSONB NOT NULL DEFAULT '[]'::jsonb,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,

    CONSTRAINT onboarding_questions_key_check
        CHECK (question_key ~ '^[a-z0-9][a-z0-9_-]{0,79}$'),
    CONSTRAINT onboarding_questions_prompt_check
        CHECK (jsonb_typeof(prompt) = 'object' AND prompt <> '{}'::jsonb),
    CONSTRAINT onboarding_questions_type_check
        CHECK (answer_type IN ('single_choice', 'multiple_choice', 'text', 'scale', 'number', 'boolean')),
    CONSTRAINT onboarding_questions_options_check
        CHECK (jsonb_typeof(options) = 'array'),
    CONSTRAINT onboarding_questions_settings_check
        CHECK (jsonb_typeof(settings) = 'object'),
    CONSTRAINT onboarding_questions_position_check
        CHECK (position >= 0),
    CONSTRAINT onboarding_questions_key_unique
        UNIQUE (test_id, question_key),
    CONSTRAINT onboarding_questions_position_unique
        UNIQUE (test_id, position)
);

CREATE TABLE onboarding_responses (
    user_id UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,
    test_slug VARCHAR(80) NOT NULL,
    test_version INTEGER NOT NULL,
    answers JSONB NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT onboarding_responses_answers_check
        CHECK (jsonb_typeof(answers) = 'object'),
    CONSTRAINT onboarding_responses_user_test_pk
        PRIMARY KEY (user_id, test_slug),
    CONSTRAINT onboarding_responses_test_fk
        FOREIGN KEY (test_slug, test_version)
        REFERENCES onboarding_tests(slug, version) ON DELETE CASCADE
);

CREATE INDEX onboarding_responses_submitted_at_idx
    ON onboarding_responses(user_id, submitted_at DESC);
