---
title: Telegram
---

You can enable Telegram for a user or admin [notifications](https://remark42.com/docs/configuration/notifications/) and user auth.

To set up notifications or auth with Telegram, first, you need to create a bot and write its access token to the remark42 configuration.

## Getting bot token for Telegram

To get a token, talk to [BotFather](https://core.telegram.org/bots#6-botfather). All you need is to send `/newbot` command and choose the name for your bot (it must end in `bot`). That is it, and you got a token which you'll need to write down into remark42 configuration as `TELEGRAM_TOKEN`.

_Example of such a "talk"_:

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

## Authentication

To enable Telegram authentication for the users, set variable `AUTH_TELEGRAM=true`.

## Notifications

### Notifications for administrators

To integrate notifications about any comment on your sites with remark42 with [Telegram](https://telegram.org)

1. Set `NOTIFY_ADMINS=telegram`
2. Make [a channel](https://telegram.org/faq_channels), **add your bot as Administrator** into it and add channel ID to remark42 configuration as `NOTIFY_TELEGRAM_CHAN`
  * "Post messages" permission is enough for bot to be able to post messages in your channel, others are unnecessary;
  * To obtain a public channel ID, forward any message from it to [@JsonDumpBot](https://t.me/JsonDumpBot): look for `id` in `forward_from_chat`;
  * If you want to use a private channel or chat, use [these instructions](https://github.com/GabrielRF/telegram-id#web-channel-id) to obtain the ID.

### Notifications for users

**IMPORTANT: It doesn't work as of 20.12.2021, will be working after [this](https://github.com/umputun/remark42/issues/830) issue resolution** (waits for frontend support).

Enabling Telegram user notifications allows users to sign up for notifications about replies to their messages. To do it, set the variable `NOTIFY_USERS=telegram`.

### Technical details

Telegram notifications formatting is [limited](https://core.telegram.org/bots/api#html-style) by Telegram API and, because of that, lose most of the formatting of the original comment. Notification implementation of the remark42 backend takes the rendered HTML of the comment and strips it of the unsupported tags before sending it to Telegram.

The only way to improve the formatting of the messages would be to replace unsupported tags with supported ones, [like](https://github.com/umputun/remark42/issues/1202) `h1`-`h6` with `<b>`.
