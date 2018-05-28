#!/bin/sh

# this scrips makes a backup file to /srv/var/userbackup-<site>-<timestamp>.gz

backup_file=${BACKUP_PATH}/userbackup-${1}-$(date +%s).gz
echo "make backup file for site $1 to $backup_file"
curl "http://127.0.0.1:8081/api/v1/admin/export?site=${1}&secret=${SECRET}" > ${backup_file}
