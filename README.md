# LASTOP GROUP — Платформа

B2B-платформа для логистической отрасли: новости, проекты, мероприятия, выставки, резюме/вакансии, каталог товаров и услуг, форум, сообщества и компании.

## Стек

- **Backend**: Go 1.25, `cmd/server/main.go` (HTTP-сервер + WebSocket для realtime)
- **БД**: PostgreSQL (через `pgx/v5`)
- **Storage**: локальная файловая система (с переключением на S3/Yandex Object Storage через `STORAGE_TYPE=s3`)
- **Frontend**: vanilla JS, без сборщиков, статика из `web/`
- **Деплой**: Railway

## Запуск локально

```bash
export DATABASE_URL='<your-postgres-connection-string>'
go run ./cmd/server
```

После старта откройте http://localhost:8080.

## Структура

- `cmd/server/main.go` — основной HTTP-сервер, ~24 500 строк
- `internal/storage/` — абстракция над файловым хранилищем (локальное / S3)
- `internal/mailer/` — отправка писем (verify email, восстановление пароля)
- `internal/captcha/` — защита от ботов
- `internal/errtrack/` — Sentry-интеграция
- `internal/metrics/` — slog/runtime метрики
- `web/` — фронтенд (45 HTML-страниц + assets)
- `scripts/` — backup/restore Postgres + миграция файлов в S3

## Переменные окружения

Основные:
- `DATABASE_URL` (обязательно)
- `PORT` (по умолчанию 8080)
- `JWT_SECRET` — секрет для подписи токенов сессий
- `STORAGE_TYPE` — `local` (по умолчанию) или `s3`
- `S3_*` — настройки S3 (см. `internal/storage/s3.go`)
- `SMTP_*` — отправка писем
- `SENTRY_DSN` — отслеживание ошибок (опционально)
- `DADATA_TOKEN` — токен Dadata для live-проверки ИНН компаний (пример: `50f2c0df6a274874fec5edd4d95c468ef31db9dc`)

См. `internal/storage/s3.go`, `internal/mailer/mailer.go` для полного списка.

## Бэкапы

Скрипты в `scripts/`:
- `backup.sh` — полный дамп Postgres в S3
- `restore.sh` — восстановление из бэкапа
- `cleanup-old-backups.sh` — удаление старых дампов
- `migrate-to-s3/` — миграция файлов из `/uploads` в S3

См. `BACKUP.md` для подробностей настройки.
<!-- codex-test: связь работает -->
