---
title: Email templating
---

Remark42 uses golang templates for email templating. Templates are located in `backend/templates` and embedded into binary by statik

For getting access to the files you can use package `templates` from `backend/app/templates`

Now we have following templates:
- `email_confirmation_login.html.tmpl` – used for confirmation of login
- `email_confirmation_subscription.html.tmpl` – used for confirmation of subscription
– `email_reply.html.tmpl` – used for sending replies to user comments (when user subscribed to it) and for noticing admins about new comments on a site
– `email_unsubscribe.html.tmpl` – used for notification about successful unsubscribe from replies
– `error_response.html.tmpl` – used for ...
