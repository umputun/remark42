---
title: Automatic and Manual Backup
---

## Automatic

Remark42 by default makes daily backup files under `${BACKUP_PATH}` (default `./var/backup`). Backups kept up to `${MAX_BACKUP_FILES}` (default 10). Each backup file contains exported and gzipped content, i.e., all comments. At any point, the user can restore such backup and revert all comments to the desired state.

**Note:** The [restore procedure](https://remark42.com/docs/backup/restore/) cleans the current data store and replaces all comments from the backup file.

## Manual

You can make a backup manually whenever you want. Run the command (`ADMIN_PASSWD` must be enabled on the server for it to work):
`docker exec -it remark42 backup -s {your site ID}`

This command creates `userbackup-{site ID}-{timestamp}.gz` file by default.

## Backup format

The backup file is a text file with all exported comments separated by EOL. Each backup record is a valid JSON with all key/value unmarshaled from the `Comment` struct (see [here](https://remark42.com/docs/contributing/api/#commenting)).
