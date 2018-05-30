#!/bin/sh

# this scrips making a backup file to /tmp/export-remark.gz and loading it back
# useful to migrate data schema in case if new version of data store incomaptible with the stored comments.
set -e
echo "make backup file for site $1"
curl "http://127.0.0.1:8081/api/v1/admin/export?site=${1}&secret=${SECRET}" > /tmp/export-remark.gz

BOLTDB_PATH=${BOLTDB_PATH:-./var}
BACKUP_PATH=${BACKUP_PATH:-./var}

cp ${BOLTDB_PATH}/${1}.db ${BACKUP_PATH}/${1}-$(date +%s).db

echo "import backup to site $1"
echo "unpack /tmp/export-remark.gz"
gunzip -c /tmp/export-remark.gz >/tmp/backup.remark
ls -laH /tmp/backup.remark

echo "export to site $1"
curl -X POST -H "Content-Type: application/json" --data-binary @/tmp/backup.remark "http://127.0.0.1:8081/api/v1/admin/import?site=${1}&provider=native&secret=${SECRET}"

rm /tmp/backup.remark
rm /tmp/export-remark.gz
