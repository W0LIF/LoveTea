#!/bin/bash
# Восстановление из резервной копии PostgreSQL для LoVeTea
# Запускать из папки infra/
# Использование: ./restore.sh backups/postgres_20261121_120000.sql.gz

set -euo pipefail

BACKUP_FILE="${1:-}"
if [ -z "$BACKUP_FILE" ] || [ ! -f "$BACKUP_FILE" ]; then
  echo "Usage: $0 <path-to-backup-file.sql.gz>"
  exit 1
fi

PG_HOST="${POSTGRES_HOST:-localhost}"
PG_PORT="${POSTGRES_PORT:-5434}"
PG_USER="${POSTGRES_USER:-lovetea}"
PG_DB="${POSTGRES_DB:-lovetea}"
PG_PASSWORD="${POSTGRES_PASSWORD:-lovetea}"

echo "=== Restoring PostgreSQL from $BACKUP_FILE ==="
PGPASSWORD="$PG_PASSWORD" gunzip -c "$BACKUP_FILE" | PGPASSWORD="$PG_PASSWORD" psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB"
echo "=== Restore complete ==="
