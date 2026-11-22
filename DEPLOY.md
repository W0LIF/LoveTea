# Развёртывание LoVeTea на Railway + сборка APK

## Часть 1: Бэкенд на Railway

### Предварительные требования
- Аккаунт на [railway.app](https://railway.app)
- GitHub репозиторий с кодом LoVeTea
- S3-совместимое хранилище (AWS S3, Cloudflare R2, MinIO на внешнем сервере) для медиафайлов
- SMTP сервер для email (опционально, для восстановления пароля)

### Шаг 1: Подключение репозитория к Railway

1. Войдите в [Railway Dashboard](https://railway.app/dashboard)
2. Нажмите **"New Project"** → **"Deploy from GitHub repo"**
3. Выберите репозиторий `W0LIF/LoveTea` (или загрузите код)
4. Railway автоматически обнаружит `infra/docker-compose.yml`

### Шаг 2: Настройка сервисов

Railway развернёт все сервисы из `docker-compose.yml`:
- **postgres** — PostgreSQL + PostGIS
- **redis** — Redis с паролем
- **minio** — объектное хранилище
- **mailpit** — тестовый SMTP
- **api** — Go API (Docker-образ из `backend/Dockerfile`)

### Шаг 3: Переменные окружения

В Railway Dashboard перейдите в каждый сервис и задайте переменные:

#### Сервис `api`

| Переменная | Значение |
|---|---|
| `JWT_SECRET` | любая строка минимум 32 символа |
| `MINIO_ENDPOINT` | `minio:9000` (внутреннее имя сервиса) |
| `MINIO_PUBLIC_ENDPOINT` | публичный URL MinIO (или `minio:9000`) |
| `MINIO_ACCESS_KEY` | ключ доступа к MinIO |
| `MINIO_SECRET_KEY` | секретный ключ к MinIO |
| `MINIO_BUCKET` | `lovetea-media` |
| `MINIO_USE_TLS` | `false` (внутри Railway нет TLS между сервисами) |
| `MAIL_HOST` | `mailpit` (или внешний SMTP) |
| `MAIL_PORT` | `1025` (или `587` для внешнего SMTP) |
| `MAIL_USER` | SMTP логин |
| `MAIL_PASS` | SMTP пароль |
| `MAIL_FROM` | `noreply@yourdomain.com` |
| `POSTGRES_SSLMODE` | `disable` |

#### Сервис `postgres`

| Переменная | Значение |
|---|---|
| `POSTGRES_DB` | `lovetea` |
| `POSTGRES_USER` | `lovetea` |
| `POSTGRES_PASSWORD` | любой пароль |

#### Сервис `redis`

| Переменная | Значение |
|---|---|
| `REDIS_PASSWORD` | любой пароль |

#### Сервис `minio`

| Переменная | Значение |
|---|---|
| `MINIO_ROOT_USER` | `minioadmin` (или свой) |
| `MINIO_ROOT_PASSWORD` | любой пароль |

### Шаг 4: Получение URL API

После деплоя Railway присвоит каждому сервису публичный домен:
- API: `https://lovetea-api-xxx.up.railway.app`

Этот URL понадобится для сборки мобильного APK.

### Шаг 5: Проверка health endpoint

```bash
curl https://lovetea-api-xxx.up.railway.app/health/live
curl https://lovetea-api-xxx.up.railway.app/health/ready
```

---

## Часть 2: Сборка APK мобильного приложения

### Требования
- Flutter SDK установлен
- Android SDK установлен
- URL вашего бэкенда на Railway

### Шаг 1: Сборка APK

Из папки `mobile/`:

```powershell
# Получить зависимости
flutter pub get

# Собрать release APK с указанием URL бэкенда
flutter build apk --release --dart-define=API_BASE_URL=https://lovetea-api-xxx.up.railway.app
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

### Альтернатива: сборка для локального тестирования

```powershell
flutter build apk --debug --dart-define=API_BASE_URL=http://10.0.2.2:8080
```

---

## Часть 3: Проверка работы

### Бэкенд
```bash
# Health check
curl https://lovetea-api-xxx.up.railway.app/health/live
curl https://lovetea-api-xxx.up.railway.app/health/ready

# Регистрация
curl -X POST https://lovetea-api-xxx.up.railway.app/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}'
```

### Мобильное приложение
1. Установите APK на устройство
2. Зарегистрируйте аккаунт
3. Создайте пару (приглашение → присоединение)
4. Проверьте синхронизацию данных между устройствами

---

## Часть 4: Резервное копирование

Railway создаёт автоматические снимки volumes для PostgreSQL и MinIO. Для настройки:

1. В Railway Dashboard перейдите в сервис `postgres`
2. Вкладка **Backups** — настройте расписание
3. Для файлового хранилища (MinIO/S3) настройте бэкап в консоли провайдера

Скрипты локального бэкапа находятся в `infra/scripts/`:
- `backup.sh` — создание бэкапа
- `restore.sh` — восстановление
- `cleanup.sh` — очистка старых бэкапов

---

## Часть 5: Устранение неполадок

| Проблема | Решение |
|---|---|
| API не стартует | Проверьте логи в Railway Dashboard → сервис api → Logs |
| Миграции не применяются | Убедитесь, что `POSTGRES_SSLMODE=disable` внутри Railway |
| Файлы не загружаются | Проверьте `MINIO_ENDPOINT=minio:9000` и ключи доступа |
| Push не работает | Настройте `FCM_ACCESS_TOKEN` и `APNS_TOPIC` в переменных окружения |
| APK не подключается к API | Проверьте `API_BASE_URL` при сборке и что устройство имеет доступ к интернету |
| Сервисы не видят друг друга | Используйте внутренние имена сервисов (postgres, redis, minio, mailpit) |
