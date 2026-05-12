#!/usr/bin/env bash
#
# LASTOP — PostgreSQL backup to S3.
#
# USAGE (with .env.backup):
#   source .env.backup && bash scripts/backup.sh
#
# REQUIRED ENV:
#   DATABASE_URL   — Postgres connection string
#   S3_BUCKET      — S3 bucket name
#   S3_ENDPOINT    — S3 endpoint URL (e.g. https://storage.yandexcloud.net)
#   S3_ACCESS_KEY  — S3 access key
#   S3_SECRET_KEY  — S3 secret key
#
# OPTIONAL ENV:
#   S3_REGION      — S3 region (default: ru-central1)
#   S3_PREFIX      — prefix in bucket (default: backups)
#
set -euo pipefail

require_env() {
  local name=$1
  if [ -z "${!name:-}" ]; then
    echo "ERROR: env variable $name is not set" >&2
    exit 1
  fi
}
require_env DATABASE_URL
require_env S3_BUCKET
require_env S3_ENDPOINT
require_env S3_ACCESS_KEY
require_env S3_SECRET_KEY

REGION="${S3_REGION:-ru-central1}"
PREFIX="${S3_PREFIX:-backups}"
TIMESTAMP=$(date -u +%Y-%m-%d-%H%M)
FILENAME="lastop-backup-${TIMESTAMP}.sql.gz"
LOCAL_PATH="/tmp/${FILENAME}"
S3_KEY="${PREFIX}/${FILENAME}"

echo "=== LASTOP backup ==="
echo "Started:   $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "S3 bucket: ${S3_BUCKET}"
echo "S3 key:    ${S3_KEY}"
echo

# 1. pg_dump + gzip
echo "[1/3] Dumping database..."
pg_dump --no-owner --no-privileges --clean --if-exists "${DATABASE_URL}" \
  | gzip > "${LOCAL_PATH}"
DUMP_SIZE=$(du -h "${LOCAL_PATH}" | cut -f1)
echo "      done, size: ${DUMP_SIZE}"

# 2. Upload to S3
echo "[2/3] Uploading to S3..."
AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
aws s3 cp "${LOCAL_PATH}" "s3://${S3_BUCKET}/${S3_KEY}" \
  --endpoint-url "${S3_ENDPOINT}" \
  --region "${REGION}"
echo "      done"

# 3. Cleanup local file
echo "[3/3] Removing local file..."
rm -f "${LOCAL_PATH}"
echo "      done"

echo
echo "OK: backup uploaded: ${S3_KEY}"
