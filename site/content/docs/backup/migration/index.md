---
title: Migration from Disqus/WordPress/Commento to Remark42
---

Remark42 supports importing comments from Disqus, WordPress, Commento, or native backup format. All imported comments have an `Imported` field set to `true`. All methods below remove existing comments from the site if they are present, please see the [restoration documentation](https://remark42.com/docs/backup/restore/) for instructions on import preserving existing comments. An import from Disqus, WordPress or Commento also removes the notification addresses users subscribed with, so existing subscriptions end; a native restore brings back the addresses stored in the backup. Blocked and verified users and posts with comments disabled are kept.

In `docker exec -it remark42 remark42 ...`, the first `remark42` is the container name and the second is the Remark42 binary inside it. The command sends the file to the server over HTTP at `REMARK_URL`, the public address in the container's environment, so the request goes through your reverse proxy and its request size limit. To send it to the server directly, add `--url=http://127.0.0.1:8080`, using the port the server listens on inside the container.

The import command only submits the job: it prints `completed, status=202` and exits while the server imports in the background. The result is in the server's log (`docker logs remark42`): `import request completed` with the number of comments, or a line with `import failed`. A second import for the same site is refused with 409 while one is running.

On Linux, the container makes its own `app` user the owner of `./var` at every start, which can stop a host user other than root from writing there. Copy a file into the container with `docker cp <file> remark42:/srv/var/<file>`, or set `APP_UID` to your host user's ID to keep `./var` writable for you.

## Initial import from Disqus

1. Disqus provides export of all comments on your site in a gzipped file. This option is available in your Moderation panel at Disqus Admin > Setup > Export. The export will be sent into a queue and then emailed to the address associated with your account once it's ready. Direct link to export will be something like `https://<shortname>.disqus.com/admin/discussions/export/`. See [importing-exporting](https://help.disqus.com/en/articles/1717199-importing-exporting) for more details
2. Copy this file to `/srv/var` in the Remark42 container. There is no need to unpack it: a file name ending in `.gz` is unpacked during the import
3. Run import command (`ADMIN_PASSWD` must be enabled on the server for it to work) - `docker exec -it remark42 remark42 import -p disqus -f /srv/var/{disqus-export-name}.xml.gz -s {your site ID}`

A commenter with a Disqus account gets a Remark42 user ID made from the Disqus username; a comment posted without an account gets one made from the name its author typed. A guest who typed a name equal to somebody's Disqus username therefore shares that account's user ID, and verifying, blocking or deleting that user in Remark42 applies to both.

The import also fetches each commenter's avatar from Disqus and stores it with the other avatars, so Remark42 serves it without linking to Disqus. Avatars are fetched before the existing comments are replaced, at about ten users a second; a commenter without a Disqus avatar, or whose avatar Disqus fails to return, is imported without one. The server needs outbound HTTPS access to `disqus.com` and `disquscdn.com` for this, and imports the comments without avatars when Disqus can't be reached.

Left out of the import: the comments of deleted threads, votes, and avatars of guests who commented without a Disqus account. Deleted comments and comments marked as spam are kept without their text and author only when they have replies, so the replies stay in their thread; otherwise they are not shown.

## Initial import from WordPress

1. Use [that instruction](https://wordpress.com/support/export/) to export comments to file using standard WordPress functionality
2. Copy this file to `/srv/var` in the Remark42 container
3. Run import command (`ADMIN_PASSWD` must be enabled on the server for it to work) - `docker exec -it remark42 remark42 import -p wordpress -f /srv/var/{wordpress-export-name}.xml -s {your site ID}`

Only approved comments are imported. Pingbacks, trackbacks and records plugins keep as comments, such as WooCommerce order notes, are left out. Avatars and votes are not imported. Registered WordPress users and guests are told apart by their display name only. Commenters with the same name share one Remark42 user ID, so verifying, blocking or deleting that user applies to all of them.

The visibility of a post is not carried over. Approved comments of every post in the export are imported, including private, draft, scheduled, trashed and password-protected posts, and Remark42 shows them like any other comment. Take such posts out of the export file before the import if their comments should stay hidden.

Comment text is shown the way WordPress shows it: blank lines start paragraphs, line breaks are kept, bare links and email addresses become links and text smilies become emoji. Markdown is not interpreted, as WordPress does not interpret it either. Quotes and dashes are curled unless `DISABLE_FANCY_TEXT_FORMATTING` is set.

## Initial import from Commento

1. Copy the exported JSON file to `/srv/var` in the Remark42 container
2. Run import command (`ADMIN_PASSWD` must be enabled on the server for it to work) - `docker exec -it remark42 remark42 import -p commento -f /srv/var/{commento-export-name}.json -s {your site ID}`

Comments waiting for moderation or flagged as spam in Commento are not imported, and neither is a comment whose author is missing from the export's list of commenters, except anonymous comments. Scores are not imported. An avatar is kept as the `http` or `https` address given in the export; any other value is dropped.

Comments are imported for the domain specified in the provided file, with `https://` prefix. To move them to a different domain or to an `http://` address, use [URL migration](https://remark42.com/docs/backup/url-migration/) with a prefix rule after the import.
