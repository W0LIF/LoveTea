# Развёртывание LoVeTea на Render + сборка APK

## Часть 1: Бэкенд на Render

### Предварительные требования
- Аккаунт на [render.com](https://render.com)
- GitHub репозиторий с кодом LoVeTea
- S3-совместимое хранилище (AWS S3, Cloudflare R2, MinIO на внешнем сервере) для медиафайлов
- SMTP сервер для email (опционально, для восстановления пароля)

### Шаг 1: Подготовка репозитория

Убедитесь, что в корне репозитория есть:
- `render.yaml` — конфигурация Render (уже создано)
- `backend/Dockerfile` — Docker-образ бэкенда (обновлён)
- `backend/cmd/api/main.go` — поддерживает `WORKER_MODE=reminders` (обновлён)

### Шаг 2: Подключение к Render

1. Войдите в [Render Dashboard](https://dashboard.render.com/)
2. Нажмите **"New +" → "Blueprint"**
3. Подключите ваш GitHub репозиторий `E:\LoveTea` (или загрузите код в GitHub)
4. Render автоматически обнаружит `render.yaml` и предложит создать:
   - **lovetea-api** — Web Service (Docker)
   - **lovetea-worker** — Background Worker (Docker) для reminders
   - **lovetea-postgres** — Managed PostgreSQL
   - **lovetea-redis** — Managed Redis

### Шаг 3: Настройка переменных окружения

В Render Dashboard перейдите в каждый сервис и задайте переменные:

#### lovetea-api (Web Service)

| Переменная | Значение |
|---|---|
| `MINIO_ENDPOINT` | Ваш S3 endpoint (например `s3.amazonaws.com` или `xxx.r2.cloudflarestorage.com`) |
| `MINIO_PUBLIC_ENDPOINT` | Публичный URL для доступа к файлам |
| `MINIO_ACCESS_KEY` | Ключ доступа к S3 |
| `MINIO_SECRET_KEY` | Секретный ключ к S3 |
| `MINIO_BUCKET` | `lovetea-media` |
| `MINIO_USE_TLS` | `true` |
| `MAIL_HOST` | SMTP хост (или пусто, если не нужен email) |
| `MAIL_PORT` | `587` |
| `MAIL_USER` | SMTP логин |
| `MAIL_PASS` | SMTP пароль |
| `MAIL_FROM` | `noreply@yourdomain.com` |

`POSTGRES_*`, `REDIS_*`, `JWT_SECRET` заполняются автоматически Render из managed сервисов.

#### lovetea-worker (Background Worker)

Те же переменные, что и для API, кроме `MINIO_*` (можно не задавать, если не нужна загрузка файлов из воркера).

### Шаг 4: Первичный деплой

1. Render соберёт Docker-образы и запустит сервисы
2. При первом старте API автоматически применит миграции базы данных
3. Проверьте health endpoint:
   ```
   https://lovetea-api.onrender.com/health/live
   https://lovetea-api.onrender.com/health/ready
   ```

### Шаг 5: Получить URL API

После деплоя URL API будет вида:
```
https://lovetea-api.onrender.com
```

Этот URL понадобится для сборки мобильного APK.

---

## Часть 2: Сборка APK мобильного приложения

### Требования
- Flutter SDK установлен (уже есть в проекте)
- Android SDK установлен (уже есть в проекте)
- URL вашего бэкенда на Render

### Шаг 1: Сборка APK

Из папки `mobile/`:

```powershell
# Получить зависимости
flutter pub get

# Собрать release APK с указанием URL бэкенда
flutter build apk --release --dart-define=API_BASE_URL=https://lovetea-api.onrender.com
```

APK будет создан по пути:
```
mobile/build/app/outputs/flutter-apk/app-release.apk
```

### Шаг 2: Установка на устройство

1. Передайте APK на Android-устройство
2. В настройках Android включите "Установка из неизвестных источников"
3. Установите APK
4. При первом запуске укажите URL бэкенда (если не задан через `--dart-define`)

### Альтернатива: сборка для тестирования на эмуляторе

```powershell
flutter build apk --debug --dart-define=API_BASE_URL=http://10.0.2.2:8080
```

---

## Часть 3: Проверка работы

### Бэкенд
```bash
# Health check
curl https://lovetea-api.onrender.com/health/live
curl https://lovetea-api.onrender.com/health/ready

# Регистрация
curl -X POST https://lovetea-api.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}'
```

### Мобильное приложение
1. Установите APK на устройство
2. Зарегистрируйте аккаунт
3. Создайте пару (приглашение → присоединение)
4. Проверьте синхронизацию данных между устройствами

---

## Часть 4: Резервное копирование (Render)

Render Managed PostgreSQL автоматически создаёт бэкапы. Для настройки:

1. В Render Dashboard перейдите в **lovetea-postgres**
2. Вкладка **Backups** — настройте расписание
3. Для файлового хранилища (S3/R2) настройте бэкап в консоли провайдера хранилища

Скрипты локального бэкапа находятся в `infra/scripts/`:
- `backup.sh` — создание бэкапа
- `restore.sh` — восстановление
- `cleanup.sh` — очистка старых бэкапов

---

## Часть 5: Устранение неполадок

| Проблема | Решение |
|---|---|
| API не стартует | Проверьте логи в Render Dashboard → lovetea-api → Logs |
| Миграции не применяются | Убедитесь, что `POSTGRES_SSLMODE=require` для managed Postgres |
| Файлы не загружаются | Проверьте `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY` |
| Push не работает | Настройте `FCM_ACCESS_TOKEN` и `APNS_TOPIC` в переменных окружения |
| APK не подключается к API | Проверьте `API_BASE_URL` при сборке и что устройство имеет доступ к интернету |
