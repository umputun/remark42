#!/bin/sh
set -e
echo "restore backup file $1 to site $2"
BACKUP_PATH=${BACKUP_PATH:-./var}
echo "unpack $1"
gunzip -c ${BACKUP_PATH}/$1 >/tmp/backup.remark

ls -la /tmp/backup.remark

curl -X POST -H "Content-Type: application/json" --data-binary @/tmp/backup.remark "http://127.0.0.1:8081/api/v1/admin/import?site=${2}&provider=native&secret=${SECRET}"
rm /tmp/backup.remark

echo "backup restored"