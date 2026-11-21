-- Seed onboarding test: relationship compatibility
INSERT INTO onboarding_tests (slug, version, title, description, published_at)
VALUES (
    'couple-compatibility',
    1,
    '{"ru": "Совместимость пары", "en": "Couple Compatibility"}',
    '{"ru": "Ответьте на вопросы, чтобы лучше понять друг друга", "en": "Answer questions to better understand each other"}',
    NOW()
);

-- Questions for couple compatibility test
INSERT INTO onboarding_questions (test_id, question_key, prompt, answer_type, position, is_required, options, settings)
SELECT id,
       'communication-style',
       '{"ru": "Как вы предпочитаете обсуждать важные вопросы?", "en": "How do you prefer to discuss important matters?"}',
       'single_choice',
       0,
       true,
       '[{"id": "a", "text": {"ru": "Сразу и открыто", "en": "Immediately and openly"}}, {"id": "b", "text": {"ru": "После обдумывания", "en": "After thinking it through"}}, {"id": "c", "text": {"ru": "В спокойной обстановке", "en": "In a calm setting"}}]'::jsonb,
       '{}'::jsonb
FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1;

INSERT INTO onboarding_questions (test_id, question_key, prompt, answer_type, position, is_required, options, settings)
SELECT id,
       'love-language',
       '{"ru": "Что для вас больше всего значит проявление заботы?", "en": "What matters most as a sign of care?"}',
       'single_choice',
       1,
       true,
       '[{"id": "a", "text": {"ru": "Слова поддержки", "en": "Words of affirmation"}}, {"id": "b", "text": {"ru": "Время вместе", "en": "Quality time"}}, {"id": "c", "text": {"ru": "Подарки", "en": "Gifts"}}, {"id": "d", "text": {"ru": "Помощь в делах", "en": "Acts of service"}}, {"id": "e", "text": {"ru": "Физическая близость", "en": "Physical touch"}}]'::jsonb,
       '{}'::jsonb
FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1;

INSERT INTO onboarding_questions (test_id, question_key, prompt, answer_type, position, is_required, options, settings)
SELECT id,
       'conflict-resolution',
       '{"ru": "Как вы обычно ведёте себя во время конфликта?", "en": "How do you usually behave during a conflict?"}',
       'single_choice',
       2,
       true,
       '[{"id": "a", "text": {"ru": "Обсуждаю сразу", "en": "Discuss immediately"}}, {"id": "b", "text": {"ru": "Нужна пауза", "en": "Need a break"}}, {"id": "c", "text": {"ru": "Ищу компромисс", "en": "Seek compromise"}}, {"id": "d", "text": {"ru": "Ухожу от противостояния", "en": "Avoid confrontation"}}]'::jsonb,
       '{}'::jsonb
FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1;

INSERT INTO onboarding_questions (test_id, question_key, prompt, answer_type, position, is_required, options, settings)
SELECT id,
       'planning-style',
       '{"ru": "Как вы относитесь к планированию будущего?", "en": "How do you feel about planning the future?"}',
       'single_choice',
       3,
       true,
       '[{"id": "a", "text": {"ru": "Детальный план", "en": "Detailed plan"}}, {"id": "b", "text": {"ru": "Общие направления", "en": "General directions"}}, {"id": "c", "text": {"ru": "Спонтанность", "en": "Spontaneity"}}]'::jsonb,
       '{}'::jsonb
FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1;

INSERT INTO onboarding_questions (test_id, question_key, prompt, answer_type, position, is_required, options, settings)
SELECT id,
       'relaxation',
       '{"ru": "Как вы любите отдыхать вместе?", "en": "How do you like to relax together?"}',
       'multiple_choice',
       4,
       true,
       '[{"id": "a", "text": {"ru": "Прогулки на природе", "en": "Nature walks"}}, {"id": "b", "text": {"ru": "Фильмы и сериалы", "en": "Movies and series"}}, {"id": "c", "text": {"ru": "Игры", "en": "Games"}}, {"id": "d", "text": {"ru": "Рестораны и кафе", "en": "Restaurants and cafes"}}, {"id": "e", "text": {"ru": "Домашний уют", "en": "Home comfort"}}]'::jsonb,
       '{}'::jsonb
FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1;
