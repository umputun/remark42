---
title: Anti-Spam
---

## How anti-spam works

Basic real-time bot protection relies on simply testing if user sent invisible input form field and then login is rejected as it's known to be a bot.

More sophisticated mechanism is discussed in the issue [#754](https://github.com/umputun/remark42/issues/754) but not yet implemented.

## How `cleanup` anti-spam works

During the `cleanup` command run, spam comments are detected and removed (if `--dry-run` parameter was not specified). Reaching score of 50 is considered as spam, and here are the scores:

- 12.5 points for each occurrence of the provided bad words (so that 4 bad words will result in 50 points)
- 10 points if user is in the provided list of bad users
- 10 points if comment has any links, and another 10 if there are more than 5 of them
- 20 points if comment score is 0 (e.g. nobody voted for it)
