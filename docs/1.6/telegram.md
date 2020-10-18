---
title: Telegram
---

## Telegram notifications

In order to integrate notifications from remark42 with the [telegram](https://telegram.org), you should make [a channel](https://telegram.org/faq_channels) and obtain a token. This token should be used as `NOTIFY_TELEGRAM_TOKEN`. You also need to set `NOTIFY_TYPE=telegram` and set `NOTIFY_TELEGRAM_CHAN` to your channel.

In order to get token "just talk to [BotFather](https://core.telegram.org/bots#6-botfather)". All you need is to send `/newbot` command, and choose the name for your bot (it must end in `bot`). This is it, you got a token.

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
