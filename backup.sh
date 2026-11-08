#!/bin/bash
# Backup harian SchoolVia. Pasang via cron, mis.:  0 2 * * * /opt/schoolvia/backup.sh
# Override via env: DB_NAME, DB_USER, BACKUP_DIR, KEEP_DAYS
DB_NAME="${DB_NAME:-simaka}"
DB_USER="${DB_USER:-postgres}"
BACKUP_DIR="${BACKUP_DIR:-/home/agentuser/backups/simaka}"
KEEP_DAYS="${KEEP_DAYS:-7}"
mkdir -p "$BACKUP_DIR"
sudo -u "$DB_USER" pg_dump "$DB_NAME" | gzip > "$BACKUP_DIR/simaka-$(date +%F).sql.gz"
find "$BACKUP_DIR" -name 'simaka-*.sql.gz' -mtime +"$KEEP_DAYS" -delete
# TODO produksi: salin juga ke lokasi offsite (rclone/S3/dll.) — jangan simpan backup hanya di satu mesin.
