#!/usr/bin/env bash
#
# LASTOP — restore PostgreSQL from S3 backup.
#
# USAGE:
#   S3_KEY=backups/lastop-backup-2026-05-06-1530.sql.gz ./scripts/restore.sh
#
# WARNING: this fully overwrites the current database (via pg_dump --clean).
# Before applying the dump, all active sessions to the target DB are terminated
# (otherwise DROP TABLE would fail).
#
# To skip the confirmation prompt:
#   FORCE=1 ./scripts/restore.sh
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
require_env S3_KEY

REGION="${S3_REGION:-ru-central1}"
LOCAL_PATH="/tmp/$(basename "${S3_KEY}")"

echo "=== LASTOP restore ==="
echo "S3 bucket: ${S3_BUCKET}"
echo "S3 key:    ${S3_KEY}"
echo "DB URL:    ${DATABASE_URL%@*}@***"
echo

# Confirmation (skip with FORCE=1)
if [ "${FORCE:-0}" != "1" ]; then
  read -p "WARNING: current database will be fully overwritten. Continue? (yes/NO) " ans
  if [ "${ans}" != "yes" ]; then
    echo "Aborted."
    exit 1
  fi
fi

# 1. Download dump from S3
echo "[1/4] Downloading from S3..."
AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}" \
AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}" \
aws s3 cp "s3://${S3_BUCKET}/${S3_KEY}" "${LOCAL_PATH}" \
  --endpoint-url "${S3_ENDPOINT}" \
  --region "${REGION}"
DUMP_SIZE=$(du -h "${LOCAL_PATH}" | cut -f1)
echo "      done, size: ${DUMP_SIZE}"

# 2. Terminate active sessions on target DB (otherwise DROP TABLE will fail)
echo "[2/4] Terminating active connections..."
psql "${DATABASE_URL}" -v ON_ERROR_STOP=0 <<'SQL' > /dev/null 2>&1 || true
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = current_database()
  AND pid <> pg_backend_pid();
SQL
echo "      done"

# 3. Unpack and apply dump
echo "[3/4] Restoring database..."
gunzip -c "${LOCAL_PATH}" | psql "${DATABASE_URL}" --set ON_ERROR_STOP=on > /dev/null
echo "      done"

# 4. Cleanup local file
echo "[4/4] Removing local file..."
rm -f "${LOCAL_PATH}"
echo "      done"

echo
echo "OK: database restored from ${S3_KEY}"
