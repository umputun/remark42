---
title: Notification
---

## Email

## Slack admin notifications

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

You also need to set `NOTIFY_ADMINS=slack` for the Slack notification to be active.

By default, the notification are sent to the `general` channel on slack. If you need another channel, you can specify it, for instance with `NOTIFY_SLACK_CHAN=random`.

```
    - NOTIFY_ADMINS=slack
    - NOTIFY_SLACK_CHAN=general
    - NOTIFY_SLACK_TOKEN=xoxb-....
```

### Verify the notifications on Slack

If all goes fine, you should be able to see the following message on your Slack notification channel:

> New comment from _author_ -> _original author_
>> [Demo | Remark42](http://127.0.0.1:8080/web/#remark42__comment-11288987987)
>> This is the comment written by _author_


## Telegram

### Telegram notifications for administrators

In order to integrate notifications about any comment on your sites with remark42 with [telegram](https://telegram.org)
1. Set `NOTIFY_ADMINS=telegram`
1. Make [a channel](https://telegram.org/faq_channels) and add it to remark42 configuration as `NOTIFY_TELEGRAM_CHAN`
1. Get a token according to the instruction below and add it to the configuration as well

### Getting token for Telegram

In order to get token "just talk to [BotFather](https://core.telegram.org/bots#6-botfather)". All you need is to send `/newbot` command, and choose the name for your bot (it must end in `bot`). This is it, you got a token which you'll need to write down into remark42 configuration as `TELEGRAM_TOKEN`.

_Example of such a "talk":_

```
Umputun:
/newbot

BotFather:
Alright, a new bot. How are we going to call it? Please choose a name for your bot.

Umputun:
example_comments

BotFather:
Good. Now let's choose a username for your bot. It must end in `bot`. Like this, for example: TetrisBot or tetris_bot.

Umputun:
example_comments_bot

BotFather:
Done! Congratulations on your new bot. You will find it at t.me/example_comments_bot. You can now add a description, about section and profile picture for your bot, see /help for a list of commands. By the way, when you've finished creating your cool bot, ping our Bot Support if you want a better username for it. Just make sure the bot is fully operational before you do this.

Use this token to access the HTTP API:
12345678:xy778Iltzsdr45tg

For a description of the Bot API, see this page: https://core.telegram.org/bots/api
```
