---
layout: home.njk
permalink: index.html
title: Remark42 – Privacy-focused lightweight commenting engine
---

<h1 class="text-center !text-4xl !md:text-5xl">Privacy-focused lightweight commenting engine</h1>

Remark42 allows you to have a self-hosted, lightweight, and simple (yet functional) comment engine, which doesn't spy on users. It can be embedded into blogs, articles or any other place where readers add comments.

* Social login via Google, Twitter, Facebook, Microsoft, GitHub, Yandex, Patreon and Telegram
* Login via email
* Optional anonymous access
* Multi-level nested comments with both tree and plain presentations
* Import from Disqus and WordPress
* Markdown support with friendly formatter toolbar
* Moderators can remove comments and block users
* Voting, pinning and verification system
* Sortable comments
* Images upload with drag-and-drop
* Extractor for recent comments, cross-post
* RSS for all comments and each post
* Telegram, Slack, email and webhook admin notifications for each new comment on your site
* Email and Telegram notifications for users so that they get notified when someone responds to their comments
* Export data to JSON with automatic backups
* No external databases, everything embedded in a single data file
* Fully dockerized and can be deployed in a single command
* A self-contained executable can be deployed directly to Linux, Windows and macOS
* Clean, lightweight and customizable UI with white and dark themes
* Multi-site mode from a single instance
* Integration with automatic SSL (direct or via reproxy)
* Privacy-focused

## Privacy

* Remark42 is trying to be very sensitive to any private or semi-private information.
* Authentication is requesting the minimal possible scope from authentication providers and all extra information returned by them is immediately dropped and not stored in any form.
* Generally, Remark42 keeps user ID, username and avatar link only. None of these fields exposed directly - ID and name hashed, avatar proxied.
* There is no tracking of any sort.
* Login mechanic uses JWT stored in a cookie (HttpOnly, secured). The second cookie (XSRF_TOKEN) is a random ID preventing CSRF.
* There is no cross-site login, i.e., user's behavior can't be analyzed across independent sites running Remark42.
* There are no third-party analytic services involved.
* Users can request all information Remark42 knows about them and receive the export in the gz file.
* Supports complete cleanup of all information related to user's activity by user's "deleteme" request.
* Cookie lifespan can be restricted to session-only.
* All potentially sensitive data stored by Remark42 hashed and encrypted.

<div class="text-right italic">— The Remark42 Team</div>
