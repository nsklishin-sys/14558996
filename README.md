# Greeting site (Go + TypeScript)

Небольшой проект: Go-сервер отдает API и статику, а страницы регистрации/входа работают с PostgreSQL.

## Запуск

```bash
export DATABASE_URL='postgresql://user:password@host:5432/dbname?sslmode=require'
go run ./cmd/server
```

`DATABASE_URL` обязателен, без него сервер завершится с ошибкой.

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
