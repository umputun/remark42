---
title: Telegram
---

## Telegram notifications for administrators

In order to integrate notifications about any comment on your sites with remark42 with [telegram](https://telegram.org)
1. Set `NOTIFY_ADMINS=telegram`
1. Make [a channel](https://telegram.org/faq_channels) and add it to remark42 configuration as `NOTIFY_TELEGRAM_CHAN`
1. Get a token according to the instruction below and add it to the configuration as well

## Getting token for Telegram

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
