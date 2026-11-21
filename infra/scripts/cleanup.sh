#!/bin/bash
# Очистка старых резервных копий (оставляет последние N дней)
# Запускать из папки infra/

set -euo pipefail

RETENTION_DAYS="${RETENTION_DAYS:-7}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"

if [ ! -d "$BACKUP_DIR" ]; then
  echo "Backup directory $BACKUP_DIR does not exist."
  exit 0
fi

echo "Cleaning backups older than $RETENTION_DAYS days from $BACKUP_DIR..."
find "$BACKUP_DIR" -type f -mtime "+$RETENTION_DAYS" -delete
echo "Cleanup complete."
