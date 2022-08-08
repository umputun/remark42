---
title: Notification
---

There are two types of notifications, "Admin" and "User" notifications. Admin notifications will forward every new comment on the site to your desired location, like email or Telegram channel. User notifications will allow users to subscribe to replies to their comments. Enabling user notifications doesn't enable them by default; for example, users need to click a button in the interface to subscribe to email notifications even if they are logged in by email.

[Email](https://remark42.com/docs/configuration/email/) and [Telegram](https://remark42.com/docs/configuration/telegram/) notifications are described on separate pages.

## Slack admin notifications

To integrate notifications from remark42 with [Slack](https://slack.com), you should create [a bot](https://slack.com/intl/en-cn/help/articles/115005265703-Create-a-bot-for-your-workspace) and obtain a token.

### Create a Slack Bot

1. Create a [Slack app](https://api.slack.com/apps/new) if you don't already have one, or select an existing app you've created.
2. Click the OAuth & Permissions tab in the left sidebar.
3. Below Bot Token Scopes, select the `chat:write`, `chat:write.public`, and `channels:read` scopes. Then click Add an OAuth Scope.
4. Scroll to the top of the page, and click on Install to workspace.
5. You should see the "_View basic information about public channels in your workspace_", "_Send Message as ..._" and "_Send messages to channels ... isn't a member of_" as the permission, then click allow.
6. You can then see the token, in the form of `xoxb-...-...-...`

### Remark42 configuration

The Slack token which you obtained before should be used as `NOTIFY_SLACK_TOKEN`.

You also need to set `NOTIFY_ADMINS=slack` for the Slack notification to be active.

By default, the notifications are sent to the `general` channel on Slack. If you need another channel, you can specify it with `NOTIFY_SLACK_CHAN=channel_name`.

```
    - NOTIFY_ADMINS=slack
    - NOTIFY_SLACK_CHAN=general
    - NOTIFY_SLACK_TOKEN=xoxb-....
```

### Verify the notifications on Slack

If all goes fine, you should be able to see the following message on your Slack notification channel:

> New comment from _author_ -> _original author_
>
> > [Demo | Remark42](http://127.0.0.1:8080/web/#remark42__comment-11288987987)
> > This is the comment written by the _author_

## WebHook admin notifications

You need to set `NOTIFY_ADMINS=webhook` to enable WebHook notifications on all new comments and set at least `NOTIFY_WEBHOOK_URL` for them to start working.

Additionally, you might want to set `NOTIFY_WEBHOOK_TEMPLATE` (which is Go Template, `{"text": "{{.Text}}"}` by default) and `NOTIFY_WEBHOOK_HEADERS`, which is HTTP header(s) in format `Header1:Value1,Header2:Value2,...`.
