# Greeting site (Go + TypeScript)

Небольшой проект: Go-сервер отдает API и статику, а страницы регистрации/входа работают с PostgreSQL.

## Запуск

```bash
export DATABASE_URL='postgresql://user:password@host:5432/dbname?sslmode=require'
go run ./cmd/server
```

Также можно подключиться без `DATABASE_URL`, через переменные:

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=postgres
export PGDATABASE=greeting_site
export PGSSLMODE=disable
go run ./cmd/server
```

После старта откройте: http://localhost:8080

## Что внутри

- `cmd/server/main.go` — HTTP-сервер, endpoint `/api/greeting`, `/health` и auth API:
  - `POST /api/auth/register` — регистрация пользователя в PostgreSQL
  - `POST /api/auth/login` — вход по email/паролю из PostgreSQL
  - `GET /health` — проверка статуса БД (возвращает `database: up`, если соединение доступно)
- `web/register.html` — UI регистрации
- `web/login.html` — UI авторизации
- `web/index.html` — стартовая страница

При старте сервер ожидает `DATABASE_URL`, подключается к PostgreSQL и автоматически создает таблицу `users` через `CREATE TABLE IF NOT EXISTS`.
