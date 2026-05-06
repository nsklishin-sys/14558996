#!/usr/bin/env bash
#
# LASTOP — восстановление PostgreSQL из резервной копии в S3.
#
# ИСПОЛЬЗОВАНИЕ:
#   S3_KEY=backups/lastop-backup-2026-05-06-1530.sql.gz ./scripts/restore.sh
#
# ВНИМАНИЕ: скрипт ПОЛНОСТЬЮ перезаписывает текущую БД (через pg_dump --clean).
# Перед запуском будет запрошено подтверждение. Для отключения подтверждения:
#   FORCE=1 ./scripts/restore.sh
#
set -euo pipefail

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
require_env S3_KEY

REGION="${S3_REGION:-ru-central1}"
LOCAL_PATH="/tmp/$(basename "${S3_KEY}")"

echo "═══ LASTOP restore ═══"
echo "S3 bucket: ${S3_BUCKET}"
echo "S3 ключ:   ${S3_KEY}"
echo "DB URL:    ${DATABASE_URL%@*}@***"
echo

# Подтверждение (если не задан FORCE=1)
if [ "${FORCE:-0}" != "1" ]; then
  read -p "ВНИМАНИЕ: текущая БД будет полностью перезаписана. Продолжить? (yes/NO) " ans
  if [ "${ans}" != "yes" ]; then
    echo "Отменено."
    exit 1
  fi
fi

# 1. Скачать дамп из S3
echo "[1/3] Скачиваем из S3..."
AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
aws s3 cp "s3://${S3_BUCKET}/${S3_KEY}" "${LOCAL_PATH}" \
  --endpoint-url "${S3_ENDPOINT}" \
  --region "${REGION}"
DUMP_SIZE=$(du -h "${LOCAL_PATH}" | cut -f1)
echo "      готово, размер: ${DUMP_SIZE}"

# 2. Распаковать и применить
echo "[2/3] Восстанавливаем БД..."
gunzip -c "${LOCAL_PATH}" | psql "${DATABASE_URL}" --set ON_ERROR_STOP=on > /dev/null
echo "      готово"

# 3. Удалить локальный файл
echo "[3/3] Чистим временные файлы..."
rm -f "${LOCAL_PATH}"
echo "      готово"

echo
echo "✓ БД успешно восстановлена из ${S3_KEY}"
