# Greeting site (Go + TypeScript)

Небольшой проект: Go-сервер отдает API и статику, а страницы регистрации/входа работают с реальной базой данных SQLite.

## Запуск

```bash
go run ./cmd/server
```

После старта откройте: http://localhost:8080

## Что внутри

- `cmd/server/main.go` — HTTP-сервер, endpoint `/api/greeting`, а также auth API:
  - `POST /api/auth/register` — регистрация пользователя в SQLite
  - `POST /api/auth/login` — вход по email/паролю из SQLite
- `web/register.html` — UI регистрации
- `web/login.html` — UI авторизации
- `web/index.html` — стартовая страница

База данных создается автоматически в `data/app.db` при первом запуске.
