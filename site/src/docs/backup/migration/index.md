---
title: Migration
---

Remark42 supports importing comments from Disqus, WordPress, Commento, or native backup format. All imported comments have an `Imported` field set to `true`. All methods below remove existing comments from they site if they are present, please use the last paragraph to preserve them.

### Initial import from Disqus

1. Disqus provides an export of all comments on your site in a gzipped file. This option is available in your Moderation panel at Disqus Admin > Setup > Export. The export will be sent into a queue and then emailed to the address associated with your account once it's ready. Direct link to export will be something like `https://<siteud>.disqus.com/admin/discussions/export/`. See [importing-exporting](https://help.disqus.com/en/articles/1717199-importing-exporting) for more details
2. Move this file to your Remark42 host within `./var` and extract, i.e. `gunzip <disqus-export-name>.xml.gz`
3. Run import command - `docker exec -it remark42 import -p disqus -f /srv/var/{disqus-export-name}.xml -s {your site ID}`

### Initial import from WordPress

1. Use [that instruction](https://wordpress.com/support/export/) to export comments to file using standard WordPress functionality
2. Move this file to your Remark42 host within `./var`
3. Run import command - `docker exec -it remark42 import -p wordpress -f /srv/var/{wordpress-export-name}.xml -s {your site ID}`

### Initial import from Commento

1. Move exported json file to your Remark42 host within `./var`
2. Run import command - `docker exec -it remark42 import -p commento -f /srv/var/{commento-export-name}.json -s {your site ID}`

### Import without removing existing comments

All methods above nuke the existing comments on the site. To preserve it you should make two backup files, one for the current remark42 content, and another for imported WP/Discuss/Commento content. The format of backups is straightforward json with EOL (json line) and can be easily constructed from multiple sources. Merge them together and restore them from the resulting file:

```shell
cat wp-export.json | grep -v '{"version":1' >> combined-export.json
docker exec -it remark42 restore -f combined-export.json -s {your site ID}
```
