# LoVeTea

Мир котёнка и волчонка — приватное кроссплатформенное приложение для пары.

## Состав проекта

- `mobile/` — Flutter-приложение (пока не реализовано).
- `backend/` — Go API на Chi.
- `docs/api/openapi.yaml` — контракт REST API.
- `docs/architecture/backend-roadmap.md` — этапы и границы дальнейшей разработки backend.
- `docs/product/LoVeTea.pdf` и `docs/product/LoVeTea_Technical_Plan_v1.docx` — продуктовые требования и дорожная карта.

Архитектура backend — модульный монолит. Используются PostgreSQL/PostGIS, Redis и Docker Compose.

## Локальный запуск backend

Требуется Docker Compose и Go 1.26+ для локальной разработки.

1. Создайте локальный файл окружения, только если его ещё нет:

   ```powershell
   if (-not (Test-Path .env)) { Copy-Item .env.example .env }
   ```

2. Задайте в `.env` уникальный `JWT_SECRET` длиной не менее 32 символов и при необходимости измените пароли.
3. Из папки `infra/` запустите API и зависимости:

   ```powershell
   docker compose up --build -d
   ```

   При старте API применяет новые миграции PostgreSQL. Данные хранятся в именованных Docker volumes.
4. Проверьте состояние:

   ```powershell
   Invoke-RestMethod http://localhost:8080/health/live
   Invoke-RestMethod http://localhost:8080/health/ready
   ```

Остановка без удаления данных: `docker compose down` (из папки `infra/`). Не используйте `docker compose down -v`, если хотите сохранить локальную базу и файлы.

Запустить Go-тесты:

```powershell
Set-Location backend
go test ./...
```

Для локального запуска API вне Docker сначала запустите PostgreSQL, Redis и MinIO из Compose (из папки `infra/`), задайте `$env:MINIO_ENDPOINT='localhost:9000'`, `$env:MINIO_PUBLIC_ENDPOINT='localhost:9000'`, затем из `backend/` выполните `go run ./cmd/api`. Переменные окружения берутся из процесса; файл `../.env` загружается как удобство для локальной разработки.

## API

Все прикладные маршруты имеют префикс `/api/v1`. Полная схема запросов и ответов находится в [docs/api/openapi.yaml](docs/api/openapi.yaml).

Основные маршруты:

- `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`
- `GET /users/me`, `PATCH /users/me`
- `GET /home`
- `GET /tasks`, `POST /tasks`, `PATCH /tasks/{taskID}`, `DELETE /tasks/{taskID}?version=...`
- `GET /calendar/events`, `POST /calendar/events`, `PUT/DELETE /calendar/events/{eventID}`
- `GET /calendar/dates`, `POST /calendar/dates`, `PUT/DELETE /calendar/dates/{dateID}`
- `GET/POST /moments`
- `GET /media`, `POST /media/uploads`, `POST /media/{mediaID}/complete`, `GET /media/{mediaID}/download`, `DELETE /media/{mediaID}?version=...`
- `GET/POST /wellness/mood`, `GET/POST /wellness/sleep`
- `GET/POST /habits`, `PUT/DELETE /habits/{habitID}`
- `GET/POST/DELETE /habits/{habitID}/completions`, `GET /habits/{habitID}/stats`
- `POST /couples`, `POST /couples/{coupleID}/invitations`, `POST /couples/join`
- `GET /couples/me`, `PATCH /couples/me/profile`, `POST /couples/me/detach/confirm`
- `GET /onboarding/tests`, `POST /onboarding/answers`, `GET /onboarding/answers`
- `GET /sync/pull?cursor=0&limit=100`, `POST /sync/push`, `GET /sync/ws` (WebSocket cursor hints)

Тесты onboarding и вопросы должны быть явно добавлены и опубликованы в БД: каталог сейчас пуст, поскольку содержание и оценивание тестов не заданы спецификацией. Ответы доступны только их владельцу; партнёрский endpoint отсутствует.

Регистрация и вход выполняются сразу по email и паролю, подтверждение email не требуется. Восстановление пароля по одноразовой ссылке с ограниченным сроком действия (`POST /auth/forgot-password`, `POST /auth/reset-password`). Доступ к защищённым маршрутам — через `Authorization: Bearer <access_token>`.

API воспоминаний временно возвращает `501`: по требованиям продуктового плана их содержимое должно быть сквозно зашифровано, а протокол и управление ключами ещё не определены. Ранее созданные записи в базе не удаляются, но через API недоступны.

WebSocket `/sync/ws` передаёт только best-effort подсказки `sync.available` с персональным курсором, без содержимого изменений. Клиент должен вызывать `/sync/pull` при подключении и переподключении; этот endpoint остаётся источником истины.

`POST /sync/push` accepts bounded task, calendar-event, important-date, habit, habit-completion, mood-entry, and sleep-entry operations. Versioned entities support optimistic upsert/delete; wellness entries support versioned upsert only. Habit completions use an idempotent `(habit_id, completed_on)` key and `base_version=0` for both marking and unmarking. Each operation has a client-generated entity ID and operation ID; identical retries replay the saved result for 30 days. Operations in a batch commit independently; if a batch is retried after a partial failure, completed items replay without applying twice. Mood notes are present only in the owner's personal sync feed; shared mood events omit the note. Other entity types still use their module REST endpoints and are not yet accepted by sync push.

Загрузка медиа ограничена корректными JPEG/PNG размером до 20 MiB: API выдаёт краткоживущую HTML form POST policy для приватного MinIO bucket, а затем декодирует и проверяет загруженный объект перед публикацией. Подписанные URL загрузки/чтения являются секретами доступа; настройте `MINIO_PUBLIC_ENDPOINT` на адрес, достижимый с клиентского устройства (для эмулятора/телефона `localhost` может быть недоступен).
