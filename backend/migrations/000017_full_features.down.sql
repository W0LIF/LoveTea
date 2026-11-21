-- Обратная миграция 000017 — удаляет все новые таблицы и столбцы.
-- Порядок важен из-за внешних ключей.

-- Удалить E2E-столбцы из memories
ALTER TABLE memories DROP COLUMN IF EXISTS content_encrypted;
ALTER TABLE memories DROP COLUMN IF EXISTS ciphertext;
ALTER TABLE memories DROP COLUMN IF EXISTS cipher_nonce;
ALTER TABLE memories DROP COLUMN IF EXISTS cipher_key_ref;

-- Удалить столбец recurrence_rule из calendar_events
ALTER TABLE calendar_events DROP COLUMN IF EXISTS recurrence_rule;

-- Остальные таблицы удаляются в обратном порядке ключей
DROP TABLE IF EXISTS reminder_log CASCADE;
DROP TABLE IF EXISTS reminder_events CASCADE;
DROP TABLE IF EXISTS account_exports CASCADE;
DROP TABLE IF EXISTS date_recommendations CASCADE;
DROP TABLE IF EXISTS date_ideas CASCADE;
DROP TABLE IF EXISTS location_geofences CASCADE;
DROP TABLE IF EXISTS location_positions CASCADE;
DROP TABLE IF EXISTS location_sharing CASCADE;
DROP TABLE IF EXISTS wheel_spins CASCADE;
DROP TABLE IF EXISTS wheel_sectors CASCADE;
DROP TABLE IF EXISTS achievement_unlocks CASCADE;
DROP TABLE IF EXISTS achievements CASCADE;
DROP TABLE IF EXISTS game_sessions CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS notification_settings CASCADE;
DROP TABLE IF EXISTS letters CASCADE;
DROP TABLE IF EXISTS chat_message_reactions CASCADE;
DROP TABLE IF EXISTS chat_messages CASCADE;
DROP TABLE IF EXISTS chat_conversations CASCADE;
