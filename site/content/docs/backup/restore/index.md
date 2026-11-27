---
title: Restore Backup
---

Restore removes all comments of the site and the notification addresses users subscribed with, then imports the file, which brings back the comments and the addresses stored in it. Blocked and verified users and posts with comments disabled are kept and updated from the file.

Restore, like backup, import and URL migration, is an admin API call. The command signs in with `ADMIN_PASSWD`, and once the password is set it works from any address that can reach the server, so set it only while you need it. Signed-in administrators can make these calls without it. The recommended way to restore from the backup is to run the `restore` command inside the container:

`docker exec -it remark42 remark42 restore -f {backup-filename.gz} -s {your site ID}`

In `docker exec -it remark42 remark42 ...`, the first `remark42` is the container name and the second is the Remark42 binary inside it. The command sends the file to the server over HTTP at `REMARK_URL`, the public address in the container's environment, so the request goes through your reverse proxy and its request size limit. To send it to the server directly, add `--url=http://127.0.0.1:8080`, using the port the server listens on inside the container.

A file name without `/` is looked up in `${BACKUP_PATH}`, `/srv/var/backup` in the container. A name ending in `.gz` is unpacked on the fly.

`-s` must be the site the backup was made from. Restore first deletes all comments of the site given with `-s`, then writes the comments of that site from the file and skips the others, which the server log reports. Restoring a backup of another site therefore deletes the comments of the site given with `-s` and writes none from the file.

The command only submits the job: it prints `completed, status=202` and exits while the server imports the file in the background. The result is in the server's log (`docker logs remark42`): `import request completed` with the number of comments, or a line with `import failed`. A second restore, import or URL migration for the same site is refused with 409 while one is running.

## Import/restore without removing existing comments

The `restore` command removes the existing comments on the site, and so does an import. To add comments from Disqus, WordPress or Commento while keeping the ones you have, combine two backups and restore the result. Stop all writes from step 1 until the server log confirms the final restore: a comment or subscription saved in between is in neither file and is lost. For example, block the site at the reverse proxy and run the commands with `--url=http://127.0.0.1:8080`, which reaches the server directly instead of through `REMARK_URL`.

1. Back up the site: `docker exec -it remark42 remark42 backup -s {your site ID} -f current.gz`
2. Import the comments from the other system as described in [Migration](https://remark42.com/docs/backup/migration/). This replaces the existing comments.
3. Back up the site again: `docker exec -it remark42 remark42 backup -s {your site ID} -f imported.gz`
4. Combine the two files, leaving out the metadata line of the second one:

   ```shell
   docker exec remark42 sh -c 'cd var/backup && { gunzip -c current.gz; gunzip -c imported.gz | tail -n +2; } > combined.json'
   ```

5. Restore the combined file: `docker exec -it remark42 remark42 restore -f combined.json -s {your site ID}`

Wait for each import or restore to finish in the server's log before running the next step.
