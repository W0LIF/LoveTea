-- Обратная миграция — удаляем комментарии, добавленные в 000016.
-- Логика удаления в Go-коде остаётся; комментарий просто убирается.
DROP COMMENT IF EXISTS ON TABLE habits;
DROP COMMENT IF EXISTS ON TABLE tasks;
DROP COMMENT IF EXISTS ON TABLE calendar_events;
DROP COMMENT IF EXISTS ON TABLE important_dates;
DROP COMMENT IF EXISTS ON TABLE mood_entries;
DROP COMMENT IF EXISTS ON TABLE sleep_entries;
DROP COMMENT IF EXISTS ON TABLE moments;
DROP COMMENT IF EXISTS ON TABLE media_files;
