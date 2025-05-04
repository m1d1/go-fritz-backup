#!/usr/bin/env bash

# Example file
# run go-fritz-backup
# cleanup older files > ROTATE_PERIOD

ROTATE_PERIOD=14
BACKUP_PATH="/opt/fritz-backup/backups"
BINARY_PATH="/opt/fritz-backup/
cd "$BINARY_PATH"
./go-fritz-backup-linux-arm

# delete files older than ${ROTATE_PERIOD} days.
find ${BACKUP_PATH} -mtime +${ROTATE_PERIOD} -delete
