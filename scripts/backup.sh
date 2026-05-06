#!/usr/bin/env bash
#
# LASTOP — снятие резервной копии PostgreSQL и заливка в S3-хранилище.
#
# ИСПОЛЬЗОВАНИЕ:
#   ./scripts/backup.sh
#
# ТРЕБУЕТ env-переменные:
#   DATABASE_URL — postgres URL (например postgresql://user:pass@host:5432/db)
#   S3_BUCKET    — имя bucket
#   S3_ENDPOINT  — endpoint URL (например https://storage.yandexcloud.net)
#   S3_REGION    — регион (например ru-central1)
#   S3_ACCESS_KEY и S3_SECRET_KEY — ключи доступа
#
# ЗАВИСИМОСТИ:
#   - pg_dump (установить: apt install postgresql-client / brew install postgresql)
#   - aws CLI (установить: pip install awscli)
#   - gzip (стандартный)
#
# ВЫХОД: 0 при успехе, ненулевой код при ошибке.
#
set -euo pipefail

# Проверка обязательных env
require_env() {
  local name=$1
  if [ -z "${!name:-}" ]; then
    echo "ОШИБКА: переменная $name не задана" >&2
    exit 1
  fi
}
require_env DATABASE_URL
require_env S3_BUCKET
require_env S3_ENDPOINT
require_env S3_ACCESS_KEY
require_env S3_SECRET_KEY

REGION="${S3_REGION:-ru-central1}"

# Имя файла бэкапа: backups/2026-05-06-1530.sql.gz
TIMESTAMP=$(date -u +"%Y-%m-%d-%H%M")
BACKUP_NAME="lastop-backup-${TIMESTAMP}.sql.gz"
S3_KEY="backups/${BACKUP_NAME}"
LOCAL_PATH="/tmp/${BACKUP_NAME}"

echo "═══ LASTOP backup ═══"
echo "Дата:        ${TIMESTAMP}"
echo "S3 bucket:   ${S3_BUCKET}"
echo "S3 endpoint: ${S3_ENDPOINT}"
echo "S3 ключ:     ${S3_KEY}"
echo

# 1. Снимаем дамп и сразу сжимаем
echo "[1/3] Снимаем дамп Postgres..."
pg_dump --no-owner --no-acl --clean --if-exists "${DATABASE_URL}" | gzip -9 > "${LOCAL_PATH}"
DUMP_SIZE=$(du -h "${LOCAL_PATH}" | cut -f1)
echo "      готово, размер: ${DUMP_SIZE}"

# 2. Заливаем в S3
echo "[2/3] Заливаем в S3..."
AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
aws s3 cp "${LOCAL_PATH}" "s3://${S3_BUCKET}/${S3_KEY}" \
  --endpoint-url "${S3_ENDPOINT}" \
  --region "${REGION}"
echo "      готово"

# 3. Удаляем локальный файл (бэкап теперь в S3)
echo "[3/3] Удаляем локальный файл..."
rm -f "${LOCAL_PATH}"
echo "      готово"

echo
echo "✓ Бэкап успешно создан: s3://${S3_BUCKET}/${S3_KEY}"
echo "  Восстановление:"
echo "  S3_KEY=${S3_KEY} ./scripts/restore.sh"
