# Notify

[![Build Status](https://github.com/go-pkgz/notify/workflows/build/badge.svg)](https://github.com/go-pkgz/notify/actions) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/notify/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/notify?branch=master) [![Go Reference](https://pkg.go.dev/badge/github.com/go-pkgz/notify.svg)](https://pkg.go.dev/github.com/go-pkgz/notify)

This library provides ability to send notifications using multiple services:

- Email
- Telegram
- Slack
- Webhook

## Install

`go get -u github.com/go-pkgz/notify`

## Usage

All supported notification methods could adhere to the following interface. Example on how to use it:

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-pkgz/notify"
)

func main() {
	// create notifiers
	notifiers := []notify.Notifier{
		notify.NewWebhook(notify.WebhookParams{}),
		notify.NewEmail(notify.SMTPParams{}),
		notify.NewSlack("token"),
	}
	tg, err := notify.NewTelegram(notify.TelegramParams{Token: "token"})
	if err == nil {
		notifiers = append(notifiers, tg)
	}
	err = notify.Send(context.Background(), notifiers, "https://example.com/webhook", "Hello, world!")
	if err != nil {
		fmt.Printf("Sent message error: %s", err))
	}
}
```

### Email

`mailto:` [scheme](https://datatracker.ietf.org/doc/html/rfc6068) is supported. Only `subject` and `from` query params are used.

Examples:

- `mailto:"John Wayne"<john@example.org>?subject=test-subj&from="Notifier"<notify@example.org>`
- `mailto:addr1@example.org,addr2@example.org?&subject=test-subj&from=notify@example.org`

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-pkgz/notify"
)

func main() {
	wh := notify.NewEmail(notify.SMTPParams{
		Host:        "localhost", // the only required field, others are optional
		Port:        25,
		TLS:         false, // TLS, but not STARTTLS
		ContentType: "text/html",
		Charset:     "UTF-8",
		Username:    "username",
		Password:    "password",
		TimeOut:     time.Second * 10, // default is 30 seconds
	})
	err := wh.Send(
		context.Background(),
		`mailto:"John Wayne"<john@example.org>?subject=test-subj&from="Notifier"<notify@example.org>`,
		"Hello, World!",
	)
	if err != nil {
		log.Fatalf("problem sending message using email, %v", err)
	}
}
```

### Telegram

`telegram:` scheme akin to `mailto:` is supported. Query params `parseMode` ([doc](https://core.telegram.org/bots/api#formatting-options), legacy `Markdown` by default, preferable use `MarkdownV2` or `HTML` instead). Examples:

- `telegram:channel`
- `telegram:channelID` // channel ID is a number, like `-1001480738202`: use [that instruction](https://remark42.com/docs/configuration/telegram/#notifications-for-administrators) to obtain it
- `telegram:userID`

[Here](https://remark42.com/docs/configuration/telegram/#getting-bot-token-for-telegram) is an instruction on obtaining token for your notification bot.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-pkgz/notify"
)

func main() {
	tg, err := notify.NewTelegram(notify.TelegramParams{
		Token:      "token",          // required
		Timeout:    time.Second * 10, // default is 5 seconds
		SuccessMsg: "Success",        // optional, for auth, set by default
		ErrorMsg:   "Error",          // optional, for auth, unset by default
	})
	if err != nil {
		log.Fatalf("problem creating telegram notifier, %v", err)
	}
	err = tg.Send(context.Background(), "telegram:-1001480738202", "Hello, World!")
	if err != nil {
		log.Fatalf("problem sending message using telegram, %v", err)
	}
}
```

#### HTML Formatting

parseMode `HTML` supports [limited set of tags](https://core.telegram.org/bots/api#html-style), so `Telegram` provides `TelegramSupportedHTML` method which strips all unsupported tags and replaces `h1-h3` with `<b>` and `h4-h6` with `<i><b>` to preserve formatting.

If you want to post text into HTML tag like <a>text</a>, you can use `EscapeTelegramText` method to escape it (by replacing symbols `&`, `<`, `>` with `&amp;`, `&lt;`, `&gt;`).

#### Authorisation

You can use Telegram notifications as described above, just to send messages. But also, you can use `Telegram` to authorise users as a login method or to sign them up for notifications. Functions used for processing updates from users are `GetBotUsername`, `AddToken`, `CheckToken`, `Request`, and `Run` or `ProcessUpdate` (only one of two can be used at a time).

Normal flow is following:
1. you run the `Run` goroutine 
2. call `AddToken` and provide user with that token
3. user clicks on the link `https://t.me/<BOT_USERNAME>/?start=<TOKEN>`
4. you call `CheckToken` to verify that the user clicked the link, and if so you will receive user's UserID

Alternative flow is the same, but instead of running the `Run` goroutine, you set up process update flow separately (to use with [auth](https://github.com/go-pkgz/auth/blob/master/provider/telegram.go) as well, for example) and run `ProcessUpdate` when update is received. Example of such a setup can be seen [in Remark42](https://github.com/umputun/remark42/blob/c027dcd/backend/app/providers/telegram.go).

### Slack

`slack:` scheme akin to `mailto:` is supported. `title`, `titleLink`, `attachmentText` and query params are used: if they are defined, message would be sent with a [text attachment](https://api.slack.com/reference/messaging/attachments). Examples:

- `slack:channel`
- `slack:channelID`
- `slack:userID`
- `slack:channel?title=title&attachmentText=test%20text&titleLink=https://example.org`

```go
package main

import (
	"context"
	"log"

	"github.com/go-pkgz/notify"
	"github.com/slack-go/slack"
)

func main() {
	wh := notify.NewSlack(
		"token",
		slack.OptionDebug(true), // optional, you can pass any slack.Options
	)
	err := wh.Send(context.Background(), "slack:general", "Hello, World!")
	if err != nil {
		log.Fatalf("problem sending message using slack, %v", err)
	}
}
```

### Webhook

`http://` and `https://` schemas are supported.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-pkgz/notify"
)

func main() {
	wh := notify.NewWebhook(notify.WebhookParams{
		Timeout: time.Second,                                          // optional, default is 5 seconds
		Headers: []string{"Content-Type:application/json,text/plain"}, // optional
	})
	err := wh.Send(context.Background(), "https://example.org/webhook", "Hello, World!")
	if err != nil {
		log.Fatalf("problem sending message using webhook, %v", err)
	}
}
```

## Status

The library extracted from [remark42](https://github.com/umputun/remark) project. The original code in production use on multiple sites and seems to work fine.

`go-pkgz/notify` library still in development and until version 1 released some breaking changes possible.
