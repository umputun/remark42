---
title: Restore Backup
---

Restore removes all existing comments before importing the contents of the supplied file.

For safety and security reasons, restore functionality is not exposed outside your server by default. The recommended way to restore from the backup is to use the provided [`backend/scripts/restore.sh`](https://github.com/umputun/remark42/blob/master/backend/scripts/restore.sh). It can run inside the container (`ADMIN_PASSWD` must be enabled on the server for it to work):

`docker exec -it remark42 restore -f {backup-filename.gz} -s {your site ID}`

### Import/restore without removing existing comments

The `restore` command nukes the existing comments on the site. You should make two backup files to preserve them, one for the current remark42 content and another for WP/Discuss/Commento content you want to import. The format of backups is plain JSON with EOL (JSON line) and can be easily constructed from multiple sources. Merge them and restore them from the resulting file:

```shell
cat wp-export.json | grep -v '{"version":1' >> combined-export.json
docker exec -it remark42 restore -f combined-export.json -s {your site ID}
```
