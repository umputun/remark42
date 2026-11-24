---
title: Automatic and Manual Backup
aliases:
  - /docs/backup/
  - /docs/backup/automatic/
  - /docs/backup/manual/
---

## Automatic

Remark42 backs up every site automatically to `${BACKUP_PATH}` (default `./var/backup`), as `backup-{site ID}-{YYYYMMDD}.gz` with the server's local date. When the server starts, it writes a backup of each site that has none for the current day, so a restart on a day that already has one writes no other, and a backup with no comments in it is not written at start when it would push an existing backup out; after that, it writes one every 24 hours while it runs. A failed backup leaves the earlier files in place; another attempt runs at the next scheduled time or the next start. The newest `${MAX_BACKUP_FILES}` (default 10) automatic backups of each site are kept. Manual backups under their default `userbackup-…` name are not counted and are never removed. Each backup file holds the site's comments and its users' and posts' settings, described in [Backup format](#backup-format), gzipped. Restoring a backup replaces the site's comments and notification addresses with the ones in the file; blocks, verified marks and disabled posts already set stay, and those in the file are added.

**Note:** The [restore procedure](https://remark42.com/docs/backup/restore/) cleans the current data store and replaces all comments from the backup file.

## Manual

You can make a backup manually whenever you want. Run the command (`ADMIN_PASSWD` must be enabled on the server for it to work):
`docker exec -it remark42 remark42 backup -s {your site ID}`

In `docker exec -it remark42 remark42 ...`, the first `remark42` is the container name and the second is the Remark42 binary inside it. The command asks the server for the backup over HTTP at `REMARK_URL`, the public address in the container's environment, so the request goes through your reverse proxy. To send it to the server directly, add `--url=http://127.0.0.1:8080`, using the port the server listens on inside the container.

This command writes `userbackup-{site ID}-{YYYYMMDDTHHMMSS}.gz` to `${BACKUP_PATH}`, which is `/srv/var/backup` in the container and `./var/backup` on the host with the usual volume. `-f` sets another file name and `-p` another directory; a file name that contains `/` is used as the full path.

## Backup format

The backup file is a text file with one JSON record per line. The first line is a metadata record, `{"version":1,"users":[…],"posts":[…]}`: the blocked and verified users, the email addresses and Telegram accounts users subscribed to notifications with, and the posts with comments disabled. Each following line is one comment, marshalled from the `Comment` struct (see [here](https://remark42.com/docs/contributing/api/#commenting)). Deleted comments and the votes of each comment are included; the hashes of the voters' IP addresses are not.

Avatars and images are stored separately. By default avatars are stored in `./var/avatars` and images in `./var/pictures` and need to be backed up, as well. For more information, check the [technical details of the backend documentation](https://remark42.com/docs/contributing/backend/#technical-details).
