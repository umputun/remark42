#!/bin/sh
echo "make backup file for site $1"
curl "http://127.0.0.1:8081/api/v1/admin/export?site=${1}&secret=${SECRET}" > /tmp/export-remark.gz

echo "import backup to site $1"
echo "unpack /tmp/export-remark.gz"
gunzip -c /tmp/export-remark.gz >/tmp/backup.remark

size=`stat -c "%s" /tmp/backup.remark`
echo "source file size ${size}"

curl -X POST -H "Content-Type: application/json" --data-binary @/tmp/backup.remark "http://127.0.0.1:8081/api/v1/admin/import?site=${1}&provider=native&secret=${SECRET}"
rm -fq /tmp/backup.remark
