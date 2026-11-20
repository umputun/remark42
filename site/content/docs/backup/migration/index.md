---
title: Migration from Disqus/WordPress/Commento to Remark42
---

Remark42 supports importing comments from Disqus, WordPress, Commento, or native backup format. All imported comments have an `Imported` field set to `true`. All methods below remove existing comments from the site if they are present, please see the [restoration documentation](https://remark42.com/docs/backup/restore/) for instructions on import preserving existing comments.

In `docker exec -it remark42 remark42 ...`, the first `remark42` is the container name and the second is the Remark42 binary inside it.

### Initial import from Disqus

1. Disqus provides export of all comments on your site in a gzipped file. This option is available in your Moderation panel at Disqus Admin > Setup > Export. The export will be sent into a queue and then emailed to the address associated with your account once it's ready. Direct link to export will be something like `https://<siteud>.disqus.com/admin/discussions/export/`. See [importing-exporting](https://help.disqus.com/en/articles/1717199-importing-exporting) for more details
2. Move this file to your Remark42 host within `./var` and extract, i.e., `gunzip <disqus-export-name>.xml.gz`
3. Run import command (`ADMIN_PASSWD` must be enabled on the server for it to work) - `docker exec -it remark42 remark42 import -p disqus -f /srv/var/{disqus-export-name}.xml -s {your site ID}`

The import also fetches each commenter's avatar from Disqus and stores it with the other avatars, so Remark42 serves it without linking to Disqus. Avatars are fetched before the existing comments are replaced, at about ten users a second; a commenter without a Disqus avatar, or whose avatar Disqus fails to return, is imported without one. The server needs outbound HTTPS access to `disqus.com` and `disquscdn.com` for this, and imports the comments without avatars when Disqus can't be reached.

### Initial import from WordPress

1. Use [that instruction](https://wordpress.com/support/export/) to export comments to file using standard WordPress functionality
2. Move this file to your Remark42 host within `./var`
3. Run import command (`ADMIN_PASSWD` must be enabled on the server for it to work) - `docker exec -it remark42 remark42 import -p wordpress -f /srv/var/{wordpress-export-name}.xml -s {your site ID}`

Only approved comments are imported. Pingbacks, trackbacks and records plugins keep as comments, such as WooCommerce order notes, are left out.

Comment text is shown the way WordPress shows it: blank lines start paragraphs, line breaks are kept, bare links and email addresses become links and text smilies become emoji. Markdown is not interpreted, as WordPress does not interpret it either. Quotes and dashes are curled unless `DISABLE_FANCY_TEXT_FORMATTING` is set.

### Initial import from Commento

1. Move exported json file to your Remark42 host within `./var`
2. Run import command (`ADMIN_PASSWD` must be enabled on the server for it to work) - `docker exec -it remark42 remark42 import -p commento -f /srv/var/{commento-export-name}.json -s {your site ID}`

Comments waiting for moderation or flagged as spam in Commento are not imported.

Comments are imported for the domain specified in the provided file, with `https://` prefix. If you want to import comments for a different domain or for `http://` domain, you'll need to export them after importing, alter the export file `url` property and re-import them.
