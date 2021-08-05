---
title: Automatic Backup
---

Remark42 by default makes daily backup files under `${BACKUP_PATH}` (default `./var/backup`). Backups kept up to `${MAX_BACKUP_FILES}` (default 10). Each backup file contains exported and gzipped content, i.e. all comments. At any point, the user can restore such backup and revert all comments to the desired state.

**Note:** Restore procedure cleans the current data store and replaces all comments with comments from the backup file.

For safety and security reasons restore functionality not exposed outside of your server by default. The recommended way to restore from the backup is to use provided `scripts/restore-backup.sh`. It can run inside the container:

`docker exec -it remark42 restore -f {backup-filename.gz} -s {your site ID}`
