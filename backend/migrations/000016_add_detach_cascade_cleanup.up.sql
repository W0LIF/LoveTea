-- Миграция 000016 — документация каскадной очистки при отвязке пары.
--
-- При подтверждении отвязки (оба участника нажали "отвязаться") удаляются
-- ВСЕ совместные данные. Личные данные (owner_user_id) остаются у каждого.
--
-- Порядок удаления важен из-за внешних ключей:
--
--   1. habit_completions  — FK -> habits(id) CASCADE
--   2. habits             — WHERE couple_id = $1 (только общие)
--   3. tasks              — WHERE couple_id = $1
--   4. calendar_events    — WHERE couple_id = $1
--   5. important_dates    — WHERE couple_id = $1
--   6. mood_entries       — WHERE shared_couple_id = $1 (только общие)
--   7. sleep_entries      — WHERE shared_couple_id = $1 (только общие)
--   8. moments            — WHERE couple_id = $1
--   9. media_files        — WHERE couple_id = $1
--  10. couple_pet_state   — WHERE couple_id = $1
--  11. sync_entity_versions — WHERE couple_id = $1
--  12. sync_changes       — WHERE couple_id = $1
--  13. realtime_outbox    — удаляется вместе с sync_changes через FK CASCADE
--
-- После очистки статус пары меняется на 'detached', left_at устанавливается
-- для всех членов, все активные приглашения отзываются.
--
-- Эта миграция создаёт только комментарий. Логика удаления реализована
-- в couples.ConfirmDetach (repository.go) в рамках транзакции.

COMMENT ON TABLE habits IS 'detach_cleanup: shared habits (couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE tasks IS 'detach_cleanup: shared tasks (couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE calendar_events IS 'detach_cleanup: shared events (couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE important_dates IS 'detach_cleanup: shared dates (couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE mood_entries IS 'detach_cleanup: shared moods (shared_couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE sleep_entries IS 'detach_cleanup: shared sleeps (shared_couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE moments IS 'detach_cleanup: shared moments (couple_id IS NOT NULL) are deleted on couple detach';
COMMENT ON TABLE media_files IS 'detach_cleanup: shared media (couple_id IS NOT NULL) are deleted on couple detach';
