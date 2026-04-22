# Greeting site (Go + TypeScript)

Небольшой стартовый проект: Go-сервер отдает API и статику, TypeScript подгружает приветственное сообщение для сайта.

## Запуск

```bash
go run ./cmd/server
```

После старта откройте: http://localhost:8080

## Что внутри

- `cmd/server/main.go` — HTTP-сервер и endpoint `/api/greeting`
- `web/index.html` — стартовая страница
- `web/src/app.ts` — TypeScript-логика загрузки приветствия
- `web/app.js` — скомпилированный JS для запуска без отдельной сборки
