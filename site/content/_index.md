---
title: Remark42 – Privacy-focused lightweight commenting engine
---

<h1 class="home__title">Privacy-focused lightweight commenting engine</h1>

Remark42 allows you to have a self-hosted, lightweight, and simple (yet functional) comment engine, which doesn't spy on users. It can be embedded into blogs, articles or any other place where readers add comments.

- Social login via Google, Facebook, Microsoft, GitHub, Apple, X (Twitter), Yandex, Patreon, Discord, Telegram and custom OAuth2 providers
- Login via email
- Optional anonymous access
- Multi-level nested comments with both tree and plain presentations
- Import from Disqus, WordPress and Commento
- Markdown support with friendly formatter toolbar
- Moderators can remove comments and block users
- Voting, pinning and verification system
- Sortable comments
- Images upload with drag-and-drop
- Extractor for recent comments, cross-post
- RSS for all comments and each post
- Telegram, Slack, email and webhook admin notifications for each new comment on your site
- Email and Telegram notifications for users so that they get notified when someone responds to their comments
- Export data to JSON with automatic backups
- No external databases, everything embedded in a single data file
- Fully dockerized and can be deployed in a single command
- A self-contained executable can be deployed directly to Linux, Windows, macOS and FreeBSD
- Clean, lightweight and customizable UI with light and dark themes
- Multi-site mode from a single instance
- Integration with automatic SSL (direct or via [Reproxy](/docs/manuals/reproxy/))
- Privacy-focused

## Privacy

Remark42 is designed to store as little personal information as it can:

- From a sign-in provider, Remark42 keeps a user ID, the display name and the avatar, and for Patreon whether the user is a paid subscriber. The built-in providers are asked for the minimal scope, and nothing else they return is stored.
- The user ID is a hash, not the provider's own identifier or the email address. The avatar is stored and served by Remark42 itself.
- With each comment and vote, Remark42 keeps a keyed hash of the IP address, never the address itself. An email address or Telegram account is kept in the database only when the user subscribes to notifications.
- Remark42 includes no analytics or tracking services.
- Login uses an HttpOnly JWT cookie, marked Secure when Remark42 runs on HTTPS, and an XSRF-TOKEN cookie, a random ID that protects against CSRF.
- There is no cross-site login: every Remark42 instance has its own sign-in and its own data, and instances exchange nothing. The same account gets the same public user ID on each, so public comments can be matched across sites.
- A signed-in user can download their comments as a gz file through the API.
- A user can request deletion of their data. Once the site administrator confirms the request, their comments, notification addresses and avatar are removed, and so are the pictures those comments show, unless another comment on the same page shows them too. The votes they gave on other comments stay, kept under their user ID.

What a site running Remark42 does with this data is up to its operator: the [privacy section of the installation guide](/docs/getting-started/installation/#privacy) lists the settings that matter. The [privacy policy](/privacy/) describes remark42.com itself.

<div class="home__signature">— The Remark42 Team</div>
