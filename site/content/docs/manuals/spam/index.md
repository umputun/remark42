---
title: Anti-Spam
---

## How anti-spam works

Remark42 does not score a comment for spam when it is posted. A filter of that kind is discussed in the issue [#754](https://github.com/umputun/remark42/issues/754) but not yet implemented. In real time the server applies the checks and limits below; the names in capitals are [parameters](https://remark42.com/docs/configuration/parameters/) of the server:

- Requests to each sign-in route under `/auth/` are limited to 2 a second from one client address.
- Email and anonymous sign-in are refused if the name is not 3 to 64 letters, digits, underscores, dots, hyphens or spaces, or starts or ends with a space. Email sign-in is also refused if the address is not a valid email address.
- `UPDATE_LIMIT` sets how many requests a second one client address can send to post or preview comments, `0.5` by default; edits and votes are counted for every comment separately.
- `RESTRICTED_WORDS` and `RESTRICTED_WORDS_FILE` list words that a new or edited comment must not contain; a comment with one of them is refused.
- `RESTRICTED_NAMES` lists names, such as the names of the site's admins, that anonymous users and email users who are not admins must not take. A user who signs in with one of them is treated as blocked.
- `READONLY_AGE` closes a post for new comments and votes that number of days after its first comment. Commenters can no longer edit their comments there either; an admin still can.

Admins can also remove comments and block users by hand, see [Admin interface](https://remark42.com/docs/manuals/admin-interface/).

## How `cleanup` anti-spam works

The `cleanup` command goes through the comments already stored on a site and removes the ones it scores as spam. It works through the API of a running server:

```shell
docker exec -it remark42 remark42 cleanup -s {your site ID} --bword=casino --bword=lottery --buser=spammer --from=20260101 --to=20260131 --dry
```

In `docker exec -it remark42 remark42 ...`, the first `remark42` is the container name and the second is the Remark42 binary inside it. Inside the container the command takes `REMARK_URL`, `SECRET` and `ADMIN_PASSWD` from the environment. Anywhere else, add `--url` with the address of the server, `--admin-passwd` and `--secret`: the binary requires a secret for every command, and `cleanup` does not use its value.

| Flag             | Description                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------------------------------- |
| `-s`, `--site`   | site ID, `remark` by default                                                                                   |
| `-w`, `--bword`  | bad word, looked up anywhere in the comment text with the letter case as given; repeat the flag for every word |
| `-u`, `--buser`  | bad user, looked up anywhere in the user ID with the letter case as given; repeat the flag for every user      |
| `--from`, `--to` | first and last day of the posts to check, as `yyyymmdd`; without them no post is left out                      |
| `--dry`          | report spam comments and remove nothing                                                                        |
| `--title`        | refresh page titles and remove nothing, see [below](#refreshing-page-titles-with-cleanup---title)              |
| `--admin-passwd` | admin password of the server, needed to remove comments; the server must have `ADMIN_PASSWD` set               |
| `--timeout`      | time limit of the run, `60m` by default                                                                        |

The dates are days in the local time of the machine that runs the command. A post is checked if its first comment is not earlier than the `--from` day and its last comment is not later than the `--to` day. The command reads at most 10,000 posts of the site.

Run the command with `--dry` first. It writes a `[SPAM]` line with the comment and its score for every comment it would remove, and ends with the number of comments checked and the number found to be spam. Run it again without `--dry` to remove them. If the server refuses to remove a comment, for example because the admin password is wrong, the command writes a warning, goes on with the other comments and ends with an error.

A comment scoring more than 50 is considered spam, and here are the scores:

- 12.5 points for each of the provided bad words found in the comment, 50 at most (so that 4 bad words will result in 50 points)
- 10 points if the user ID contains one of the provided bad users
- 10 points if the comment text contains `http:` or `https:`, and another 10 if it contains `href` five or more times
- 20 points if comment score is 0 (e.g. nobody voted for it)

The signs other than bad words add up to 50 at most, so a comment without any of the bad words is never spam, and bad words alone never pass the threshold either. One bad word needs the zero score and two of the 10-point signs; two need the zero score and one 10-point sign, or all three 10-point signs; three need the zero score or two 10-point signs; four or more need any one other sign.

## Refreshing page titles with `cleanup --title`

Comments can store the title of their page, which RSS feeds and notifications show when it is there. With `--title`, the `cleanup` command removes nothing: for every comment on the posts selected by `--from` and `--to`, it asks the server to fetch the page title and store it on the comment. The server keeps a fetched title, and a failed fetch, in a cache for 15 minutes. After a page title has changed, run the command once that time has passed. If a page cannot be read or has no title, its comments keep the titles they have. Set an admin password on the server and give the same one to `cleanup` through `ADMIN_PASSWD` or `--admin-passwd`. With `--dry`, titles stay as they are. If a title update fails, for example because the admin password is wrong or a page cannot be read, the command logs a warning, continues with the remaining comments and ends with an error.
