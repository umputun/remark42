#!/bin/sh
set -e
echo "import backup file $1 to site $2"

echo "unpack $1"
gunzip -c /srv/var/backup/$1 >/tmp/backup.remark

size=`stat -c "%s" /tmp/backup.remark`
echo "source file size ${size}"

curl -X POST -H "Content-Type: application/json" --data-binary @/tmp/backup.remark "http://127.0.0.1:8081/api/v1/admin/import?site=${2}&provider=native&secret=${SECRET}"
rm /tmp/backup.remark
