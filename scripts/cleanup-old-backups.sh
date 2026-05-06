#!/usr/bin/env bash
#
# LASTOP — удаление дампов старше KEEP_DAYS дней из S3.
# Запускается по cron'у раз в неделю/день рядом с backup.sh.
#
# ИСПОЛЬЗОВАНИЕ:
#   KEEP_DAYS=30 ./scripts/cleanup-old-backups.sh
#
# Без KEEP_DAYS — по умолчанию 30 дней.
#
set -euo pipefail

require_env() {
  local name=$1
  if [ -z "${!name:-}" ]; then
    echo "ОШИБКА: переменная $name не задана" >&2
    exit 1
  fi
}
require_env S3_BUCKET
require_env S3_ENDPOINT
require_env S3_ACCESS_KEY
require_env S3_SECRET_KEY

KEEP_DAYS="${KEEP_DAYS:-30}"
REGION="${S3_REGION:-ru-central1}"

echo "═══ LASTOP cleanup ═══"
echo "Удаляем бэкапы старше ${KEEP_DAYS} дней"
echo

# Граница даты в формате YYYY-MM-DD (UTC)
if [ "$(uname)" = "Darwin" ]; then
  CUTOFF=$(date -u -v-"${KEEP_DAYS}"d +"%Y-%m-%d")
else
  CUTOFF=$(date -u -d "${KEEP_DAYS} days ago" +"%Y-%m-%d")
fi
echo "Граница: ${CUTOFF}"
echo

# Получаем список бэкапов
LIST=$(AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
  AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
  aws s3api list-objects-v2 \
    --bucket "${S3_BUCKET}" \
    --prefix "backups/" \
    --endpoint-url "${S3_ENDPOINT}" \
    --region "${REGION}" \
    --query "Contents[].Key" \
    --output text)

if [ -z "${LIST}" ] || [ "${LIST}" = "None" ]; then
  echo "Бэкапов в S3 не найдено."
  exit 0
fi

DELETED=0
TOTAL=0
for KEY in ${LIST}; do
  TOTAL=$((TOTAL+1))
  # Извлекаем дату из имени: backups/lastop-backup-2026-05-06-1530.sql.gz → 2026-05-06
  DATE=$(echo "${KEY}" | sed -E 's|.*lastop-backup-([0-9]{4}-[0-9]{2}-[0-9]{2})-.*|\1|')
  if [ -z "${DATE}" ] || [ "${DATE}" = "${KEY}" ]; then
    echo "  [skip] не распознана дата в имени: ${KEY}"
    continue
  fi
  if [[ "${DATE}" < "${CUTOFF}" ]]; then
    echo "  [del] ${KEY} (от ${DATE})"
    AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
    AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
    aws s3 rm "s3://${S3_BUCKET}/${KEY}" \
      --endpoint-url "${S3_ENDPOINT}" \
      --region "${REGION}" > /dev/null
    DELETED=$((DELETED+1))
  fi
done

echo
echo "✓ Готово. Всего бэкапов: ${TOTAL}, удалено: ${DELETED}"
