#!/bin/bash
# Резервное копирование PostgreSQL и MinIO для LoVeTea
# Запускать из папки infra/

set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
PG_HOST="${POSTGRES_HOST:-localhost}"
PG_PORT="${POSTGRES_PORT:-5434}"
PG_USER="${POSTGRES_USER:-lovetea}"
PG_DB="${POSTGRES_DB:-lovetea}"
PG_PASSWORD="${POSTGRES_PASSWORD:-lovetea}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"
MINIO_BUCKET="${MINIO_BUCKET:-lovetea-media}"

mkdir -p "$BACKUP_DIR"

echo "=== LoVeTea Backup $TIMESTAMP ==="

# PostgreSQL backup
echo "Backing up PostgreSQL..."
PGPASSWORD="$PG_PASSWORD" pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" \
  | gzip > "$BACKUP_DIR/postgres_$TIMESTAMP.sql.gz"
echo "PostgreSQL backup: $BACKUP_DIR/postgres_$TIMESTAMP.sql.gz"

# MinIO backup (using mc if available, otherwise just note it)
if command -v mc &> /dev/null; then
  echo "Backing up MinIO bucket $MINIO_BUCKET..."
  mc alias set lovetea-minio "http://$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" 2>/dev/null || true
  mc cp --recursive "lovetea-minio/$MINIO_BUCKET" "$BACKUP_DIR/minio_$TIMESTAMP/" 2>/dev/null || \
    echo "MinIO backup skipped (mc not configured or bucket empty)"
else
  echo "MinIO backup skipped (mc not installed). Install mc (MinIO Client) for object storage backups."
fi

echo "=== Backup complete ==="
echo "Files in $BACKUP_DIR:"
ls -lh "$BACKUP_DIR" | grep "$TIMESTAMP" || true
