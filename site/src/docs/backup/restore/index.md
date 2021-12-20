---
title: Restore Backup
---

Restore will clean all comments first and then process with complete import from a given file.

`docker exec -it remark42 restore -f {backup file name} -s {your site ID}`

### Import/restore without removing existing comments

All methods above nuke the existing comments on the site. You should make two backup files to preserve them, one for the current remark42 content and another for imported WP/Discuss/Commento content. The format of backups is plain JSON with EOL (JSON line) and can be easily constructed from multiple sources. Merge them and restore them from the resulting file:

```shell
cat wp-export.json | grep -v '{"version":1' >> combined-export.json
docker exec -it remark42 restore -f combined-export.json -s {your site ID}
```
