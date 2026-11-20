---
title: Anti-Spam
---

## How anti-spam works

Basic real-time bot protection relies on simply testing if user sent invisible input form field and then login is rejected as it's known to be a bot.

More sophisticated mechanism is discussed in the issue [#754](https://github.com/umputun/remark42/issues/754) but not yet implemented.

## How `cleanup` anti-spam works

During the `cleanup` command run, spam comments are detected and removed (if `--dry` parameter was not specified). A comment scoring more than 50 is considered spam, and here are the scores:

- 12.5 points for each of the provided bad words found in the comment, 50 at most (so that 4 bad words will result in 50 points)
- 10 points if the user ID contains one of the provided bad users
- 10 points if the comment text contains `http:` or `https:`, and another 10 if it contains `href` five or more times
- 20 points if comment score is 0 (e.g. nobody voted for it)

The signs other than bad words add up to 50 at most, so a comment without any of the bad words is never spam, and bad words alone never pass the threshold either. One bad word needs the zero score and two of the 10-point signs; two need the zero score and one 10-point sign, or all three 10-point signs; three need the zero score or two 10-point signs; four or more need any one other sign.
