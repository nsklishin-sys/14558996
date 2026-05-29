# Резервное копирование БД — LASTOP

## Зачем

Postgres у Railway настроен с автоматическими бэкапами на их стороне, но:
1. При проблемах с биллингом / закрытии аккаунта они исчезают вместе со всем
2. Их нельзя скачать как файл и хранить вне Railway
3. Единственный реальный способ восстановления — это вы сами с дампом, который вы хранили

Поэтому делаем независимые бэкапы в S3 (Yandex Object Storage / R2 / B2 — что угодно).

## Зависимости

На машине, где вы запускаете backup.sh:

- `pg_dump`, `psql` (Postgres клиент): `apt install postgresql-client` (Ubuntu) / `brew install postgresql` (macOS)
- `aws` CLI: `pip install awscli` или `brew install awscli`
- `gzip` (везде есть)

Проверка:
```bash
pg_dump --version
aws --version
```

## Конфигурация

Создайте файл `.env.backup` (НЕ коммитить!) с переменными:

```bash
export DATABASE_URL=<your-postgres-connection-string>
export S3_BUCKET=lastop-backups
export S3_ENDPOINT=https://storage.yandexcloud.net
export S3_REGION=ru-central1
export S3_ACCESS_KEY=YCAJ...
export S3_SECRET_KEY=YC...
```

`DATABASE_URL` берёте из Railway Variables вкладка → нажать на password чтобы раскрыть.

## Создание бэкапа вручную

```bash
source .env.backup
./scripts/backup.sh
```

Дамп окажется в S3 по ключу `backups/lastop-backup-YYYY-MM-DD-HHMM.sql.gz`.

## Восстановление из бэкапа

```bash
source .env.backup
S3_KEY=backups/lastop-backup-2026-05-06-1530.sql.gz ./scripts/restore.sh
```

Скрипт спросит подтверждение. **БД будет полностью перезаписана.**

Для пропуска подтверждения (только для автотестов на staging):
```bash
FORCE=1 S3_KEY=... ./scripts/restore.sh
```

## Список бэкапов в S3

```bash
AWS_ACCESS_KEY_ID="$S3_ACCESS_KEY" AWS_SECRET_ACCESS_KEY="$S3_SECRET_KEY" \
  aws s3 ls "s3://${S3_BUCKET}/backups/" \
    --endpoint-url "${S3_ENDPOINT}" \
    --region "${S3_REGION}"
```

## Расписание (рекомендация)

Настройте cron на удалённой машине (VPS / домашний сервер / NAS):

```cron
# Ежедневный бэкап в 03:00 UTC
0 3 * * * cd /path/to/lastop && source .env.backup && ./scripts/backup.sh >> /var/log/lastop-backup.log 2>&1

# Чистка старых раз в неделю (хранение 30 дней)
0 4 * * 0 cd /path/to/lastop && source .env.backup && KEEP_DAYS=30 ./scripts/cleanup-old-backups.sh >> /var/log/lastop-backup.log 2>&1
```

## Альтернатива — Railway Cron

Если у вас платный план Railway, можно создать отдельный cron-сервис в Railway:

1. Создайте новый сервис в проекте Railway
2. Тип: Cron job
3. Команда: `apt-get update && apt-get install -y postgresql-client awscli && bash scripts/backup.sh`
4. Расписание: `0 3 * * *`
5. В Variables скопируйте переменные `DATABASE_URL`, `S3_*`

Минус: каждый запуск ставит зависимости заново (~30 секунд). Можно сделать кастомный Dockerfile с предустановленными `pg_dump` и `aws`.

## Что делать если основной деплой упал

1. Подготовить новый Postgres (Railway Database / любой провайдер)
2. Указать его `DATABASE_URL` в `.env.backup`
3. Запустить `S3_KEY=<последний-доступный-дамп> ./scripts/restore.sh`
4. Поднять приложение, оно подключится к восстановленной БД

## Проверка целостности

Раз в месяц рекомендуется делать проверочный restore на тестовый Postgres и убедиться что приложение поднимается. Это единственный способ убедиться что бэкапы реально работают, а не только создаются.
