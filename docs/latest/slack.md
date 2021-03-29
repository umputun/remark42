---
title: Slack
---

## Slack notifications

In order to integrate notifications from remark42 with the [slack](https://slack.com), you should create [a bot](https://slack.com/intl/en-cn/help/articles/115005265703-Create-a-bot-for-your-workspace) and obtain a token. 

### Create a Slack Bot

1. Create a [Slack app](https://api.slack.com/apps/new) if you don't already have one, or select an existing app you've created.
2. Click the OAuth & Permissions tab in the left sidebar.
3. Below Bot Token Scopes, select the `chat:write`, `chat:write.public` and `channels:read` scopes. Then click Add an OAuth Scope.
4. Scroll to the top of the page, and click on Install to workspace.
5. You should see the "_View basic information about public channels in your workspace_",  "_Send Message as ..._" and "_Send messages to channels ... isn't a member of_" as the permission, then click allow. 
6. You can then see you token, in the form of `xoxb-...-...-...`


### Remark42 configuration

The slack token which you obtained before should be used as `NOTIFY_SLACK_TOKEN`. 

You also need to set `NOTIFY_TYPE=slack` for the slack notification to be active.

By default, the notification are sent to the `general` channel on slack. If you need another channel, you can specify it, for instance with `NOTIFY_SLACK_CHAN=random`.

```
    - NOTIFY_TYPE=slack
    - NOTIFY_SLACK_CHAN=general
    - NOTIFY_SLACK_TOKEN=xoxb-....
```


### Verify the notifications on Slack 

If all goes fine, you should be able to see the following message on your slack notification channel:

> New comment from _author_ -> _original author_
>> [Demo | Remark42](http://127.0.0.1:8080/web/#remark42__comment-11288987987)
>> This is the comment written by _author_
