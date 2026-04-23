# Greeting site (Go + TypeScript)

Небольшой проект: Go-сервер отдает API и статику, а страницы регистрации/входа работают с PostgreSQL.

## Запуск

```bash
export DATABASE_URL='postgresql://user:password@host:5432/dbname?sslmode=require'
go run ./cmd/server
```

Также можно запускать сервер вообще без `DATABASE_URL`:

```bash
go run ./cmd/server
```

В текущей конфигурации при отсутствии `DATABASE_URL` сервер использует Railway PostgreSQL по умолчанию:

`postgresql://postgres:QHkIHPzHfSeSKkQnEDFkJmjQJSpUpXpb@shinkansen.proxy.rlwy.net:19703/railway`

После старта откройте: http://localhost:8080

## Что внутри

- `cmd/server/main.go` — HTTP-сервер, endpoint `/api/greeting`, `/health` и auth API:
  - `POST /api/auth/register` — регистрация пользователя в PostgreSQL
  - `POST /api/auth/login` — вход по email/паролю из PostgreSQL
  - `GET /health` — проверка статуса БД (возвращает `database: up`, если соединение доступно)
- `web/register.html` — UI регистрации
- `web/login.html` — UI авторизации
- `web/index.html` — стартовая страница

При старте сервер подключается только к PostgreSQL (SQLite полностью не используется) и автоматически создает таблицу `users` через `CREATE TABLE IF NOT EXISTS`.
