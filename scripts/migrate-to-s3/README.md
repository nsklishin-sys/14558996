# migrate-to-s3

Одноразовая утилита для переноса локальных файлов из `/data/uploads` в S3-совместимое хранилище (Yandex Object Storage / R2 / B2 / AWS S3).

## Запуск

Сначала dry-run чтобы увидеть что будет залито:

```bash
STORAGE_TYPE=s3 \
S3_BUCKET=lastop-uploads \
S3_ENDPOINT=https://storage.yandexcloud.net \
S3_REGION=ru-central1 \
S3_ACCESS_KEY=YCAJ... \
S3_SECRET_KEY=YC... \
S3_PUBLIC_URL_BASE=https://lastop-uploads.storage.yandexcloud.net \
go run ./scripts/migrate-to-s3 --dry-run --dir /data/uploads -v
```

Реальная заливка — без флага `--dry-run`.

После успешной заливки локальные файлы НЕ удаляются автоматически — удалите вручную после проверки.
