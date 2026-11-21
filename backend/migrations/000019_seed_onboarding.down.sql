DELETE FROM onboarding_questions WHERE test_id IN (
    SELECT id FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1
);
DELETE FROM onboarding_tests WHERE slug = 'couple-compatibility' AND version = 1;
