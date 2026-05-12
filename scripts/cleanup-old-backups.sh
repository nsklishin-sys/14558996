#!/usr/bin/env bash
#
# LASTOP — remove old backups from S3.
# Keeps backups for last N days (default: 30).
#
# USAGE:
#   KEEP_DAYS=30 ./scripts/cleanup-old-backups.sh
#
set -euo pipefail

require_env() {
  local name=$1
  if [ -z "${!name:-}" ]; then
    echo "ERROR: env variable $name is not set" >&2
    exit 1
  fi
}
require_env S3_BUCKET
require_env S3_ENDPOINT
require_env S3_ACCESS_KEY
require_env S3_SECRET_KEY

REGION="${S3_REGION:-ru-central1}"
PREFIX="${S3_PREFIX:-backups}"
KEEP_DAYS="${KEEP_DAYS:-30}"

CUTOFF_DATE=$(date -u -d "${KEEP_DAYS} days ago" +%Y-%m-%d 2>/dev/null || date -u -v-"${KEEP_DAYS}"d +%Y-%m-%d)

echo "=== LASTOP cleanup ==="
echo "Cutoff: keeping backups newer than ${CUTOFF_DATE}"
echo

DELETED=0
AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
aws s3 ls "s3://${S3_BUCKET}/${PREFIX}/" \
  --endpoint-url "${S3_ENDPOINT}" \
  --region "${REGION}" \
  | while read -r line; do
    FILE_DATE=$(echo "$line" | awk '{print $1}')
    FILE_NAME=$(echo "$line" | awk '{print $NF}')
    if [ -z "${FILE_NAME}" ] || [ "${FILE_NAME}" = "PRE" ]; then
      continue
    fi
    if [[ "${FILE_DATE}" < "${CUTOFF_DATE}" ]]; then
      echo "  delete: ${FILE_NAME} (${FILE_DATE})"
      AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
      AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
      aws s3 rm "s3://${S3_BUCKET}/${PREFIX}/${FILE_NAME}" \
        --endpoint-url "${S3_ENDPOINT}" \
        --region "${REGION}" > /dev/null
      DELETED=$((DELETED+1))
    fi
done

echo
echo "OK: cleanup done"
