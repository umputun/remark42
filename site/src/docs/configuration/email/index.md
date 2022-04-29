---
title: Email Settings
---

## Overview

This documentation describes how to enable the email-related capabilities of Remark.

- email authentication for users:

  enabling this will let the user log in using their emails:

  ![Email authentication](images/email_auth.png)

- email notifications for any users except anonymous:

  GitHub or Google or Twitter or any other kind of user gets the ability to get email notifications about new replies to their comments (and any of the replies down the tree):

  ![Email notifications subscription](images/email_notifications.png)

## Setup email server connection

To enable any email functionality, you need to set up an email (SMTP) server connection using these variables:

```
SMTP_HOST
SMTP_PORT
SMTP_TLS
SMTP_STARTTLS
SMTP_USERNAME
SMTP_PASSWORD
SMTP_TIMEOUT
```

## Setup email notifications

### User notifications

Here is the list of variables that affect user email notifications:

```yaml
NOTIFY_USERS=email
NOTIFY_EMAIL_FROM=notify@example.com
NOTIFY_EMAIL_VERIFICATION_SUBJ # "Email verification" by default
```

### Admin notifications

Here is the list of variables that affect admin notifications, which will be sent for each new comment on your site:

```yaml
NOTIFY_ADMINS=email
NOTIFY_EMAIL_FROM=notify@example.com
ADMIN_SHARED_EMAIL=admin@example.com
```

### Mailgun

This is an example of a configuration using [Mailgun](https://www.mailgun.com/) email service:

```yaml
- SMTP_HOST=smtp.eu.mailgun.org
- SMTP_PORT=465
- SMTP_TLS=true
- SMTP_USERNAME=postmaster@mg.example.com
- SMTP_PASSWORD=secretpassword
- AUTH_EMAIL_FROM=notify@example.com
- NOTIFY_EMAIL_FROM=notify@example.com
```

### SendGrid

This is an example of a configuration using [SendGrid](https://sendgrid.com/) email service:

```yaml
- SMTP_HOST=smtp.sendgrid.net
- SMTP_PORT=465
- SMTP_TLS=true
- SMTP_USERNAME=apikey
- SMTP_PASSWORD=key-123456789
- AUTH_EMAIL_FROM=notify@example.com
- NOTIFY_EMAIL_FROM=notify@example.com
```

### Mailgun or SendGrid without exposing your server's IP

When you don't want to expose your IP (which is impossible with any SMTP provider) and when connecting to an external SMTP server is impossible due to firewall settings setting up an SMTP-to-API bridge and sending messages through it.

To use any of the containers below within remark42 environment, set the following two `SMTP` variables:

```yaml
- SMTP_HOST=mail
- SMTP_PORT=25
```

#### stevenolen/mailgun-smtp-server

Here is `docker-compose.yml` configuration part spinning up a container for
[stevenolen/mailgun-smtp-server](https://hub.docker.com/r/stevenolen/mailgun-smtp-server):

```yaml
mailgun:
  image: stevenolen/mailgun-smtp-server
  container_name: "mail"
  hostname: "mail"

  logging:
    driver: json-file
    options:
      max-size: "10m"
      max-file: "5"

  environment:
    - MG_KEY=key-123456789
    - MG_DOMAIN=example.com
```

Please note that before [stevenolen/mailgun-smtp-server#5](https://github.com/stevenolen/mailgun-smtp-server/issues/5) is fixed, Europe domain names are not supported by this tool.


#### fgribreau/smtp-to-sendgrid-gateway

Here is `docker-compose.yml` configuration part spinning up a container for
[fgribreau/smtp-to-sendgrid-gateway](https://hub.docker.com/r/fgribreau/smtp-to-sendgrid-gateway):

```yaml
sendgrid:
  image: fgribreau/smtp-to-sendgrid-gateway
  container_name: "mail"
  hostname: "mail"

  logging:
    driver: json-file
    options:
      max-size: "10m"
      max-file: "5"

  environment:
    - SENDGRID_API=key-123456789
```

### Gmail

Configuration example for Gmail:

```yaml
- SMTP_HOST=smtp.gmail.com
- SMTP_PORT=465
- SMTP_TLS=true
- SMTP_USERNAME=example.user@gmail.com
- SMTP_PASSWORD=secretpassword
- AUTH_EMAIL_FROM=example.user@gmail.com
- NOTIFY_EMAIL_FROM=example.user@gmail.com
```

### Amazon SES

Configuration example for [Amazon SES](https://aws.amazon.com/ses/) (us-east-1 region):

```yaml
- SMTP_HOST=email-smtp.us-east-1.amazonaws.com
- SMTP_PORT=465
- SMTP_TLS=true
- SMTP_USERNAME=access_key_id
- SMTP_PASSWORD=secret_access_key
- AUTH_EMAIL_FROM=notify@example.com
- NOTIFY_EMAIL_FROM=notify@example.com
```

A domain or an email that will be used in `AUTH_EMAIL_FROM` or `NOTIFY_EMAIL_FROM` must first be [verified](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-domain-procedure.html).

[SMTP Credentials](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html) must first be obtained from [Amazon SES Console](https://console.aws.amazon.com/ses/home?region=us-east-1#smtp-settings:)

## Setup email authentication

Here is the list of variables that affect email authentication:

```
AUTH_EMAIL_ENABLE
AUTH_EMAIL_FROM
AUTH_EMAIL_SUBJ
AUTH_EMAIL_CONTENT_TYPE
AUTH_EMAIL_TEMPLATE
```

After you set `SMTP_` variables, you can allow email authentication by setting these two variables:

```yaml
- AUTH_EMAIL_ENABLE=true
- AUTH_EMAIL_FROM=notify@example.com
```

Usually, you don't need to change/set anything else. In case if you want to use a different email template, set `AUTH_EMAIL_TEMPLATE`, for instance `- AUTH_EMAIL_TEMPLATE="Confirmation email, token: {{.Token}}"`. See [verified-authentication](https://github.com/go-pkgz/auth#verified-authentication) for more details.

## Email messages templates

Remark42 uses golang templates for email templating. Templates are located in `backend/templates` and embedded into binary by statik

For getting access to the files, you can use package `templates` from `backend/app/templates`

Now we have the following templates:

- `email_confirmation_login.html.tmpl` – used for confirmation of login
- `email_confirmation_subscription.html.tmpl` – used for confirmation of subscription
- `email_reply.html.tmpl` – used for sending replies to user comments (when the user subscribed to it) and for noticing admins about new comments on a site
- `email_unsubscribe.html.tmpl` – used for notification about successful unsubscribe from replies
- `error_response.html.tmpl` – used for HTML errors
