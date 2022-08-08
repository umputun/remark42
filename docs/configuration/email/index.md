---
title: Email Settings
---

## Overview

This documentation describes how to enable the email-related capabilities of Remark.

- email authentication for users:

  enabling this will let the user log in using their emails:

  ![Email authentication](images/email_auth.png)

- email notifications for any users except anonymous:

  GitHub or Google or Twitter or any other kind of user gets the ability to get email notifications about new replies to their comments (and any of the responses down the tree):

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

Admin would receive a message for each new comment on your site. Here is the list of variables that affect them:

```yaml
NOTIFY_ADMINS=email
NOTIFY_EMAIL_FROM=notify@example.com
ADMIN_SHARED_EMAIL=admin@example.com
```

### Mailgun

Here is an example of a configuration using the [Mailgun](https://www.mailgun.com/) email service:

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

Here is an example of a configuration using the [SendGrid](https://sendgrid.com/) email service:

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

When you don't want to expose your IP (which is impossible with any SMTP provider) or when connecting to an external SMTP server is impossible due to firewall settings, set up an SMTP-to-API bridge and send messages through it.

To use any of the containers below within the Remark42 environment, set the following two `SMTP` variables:

```yaml
- SMTP_HOST=mail
- SMTP_PORT=25
```

#### stevenolen/mailgun-smtp-server

Here is the `docker-compose.yml` configuration part spinning up a container for
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

Please note that this Docker image is unmaintained and Europe domain names are not supported by this tool.

#### fgribreau/smtp-to-sendgrid-gateway

Here is the `docker-compose.yml` configuration part spinning up a container for
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

You must first [verify](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-domain-procedure.html) a domain or email you will use in `AUTH_EMAIL_FROM` or `NOTIFY_EMAIL_FROM`.

Then you should obtain [SMTP Credentials](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html) from [Amazon SES Console](https://console.aws.amazon.com/ses/home?region=us-east-1#/account).

## Setup email authentication

Here is the list of variables that affect email authentication:

```
AUTH_EMAIL_ENABLE
AUTH_EMAIL_FROM
AUTH_EMAIL_SUBJ ("remark42 confirmation" by default)
AUTH_EMAIL_CONTENT_TYPE ("text/html" by default)
```

After you set `SMTP_` variables, you can allow email authentication by setting the first two:

```yaml
- AUTH_EMAIL_ENABLE=true
- AUTH_EMAIL_FROM=notify@example.com
```

## HTML templates for emails and error messages

Remark42 uses golang templates for email templating. Templates are located in `backend/app/templates/static` and embedded into binary by `go:embed` [directive](https://pkg.go.dev/embed).

Now we have the following templates:

- `email_confirmation_login.html.tmpl` – used for confirmation of login
- `email_confirmation_subscription.html.tmpl` – used for confirmation of subscription
- `email_reply.html.tmpl` – used for sending replies to user comments (when the user subscribed to it) and for noticing admins about new comments on a site
- `email_unsubscribe.html.tmpl` – used for notification about successful unsubscribing from replies
- `error_response.html.tmpl` – used for HTML errors

To replace any template, add the file with the same name to the directory with the remark42 executable file. In case you run Remark42 inside docker-compose, you can put customised templates into a directory like `customised_templates` and then mount it like that:

```yaml
    volumes:
      - ./var:/srv/var
      - ./customised_templates/email_confirmation_login.html.tmpl:/srv/email_confirmation_login.html.tmpl:ro
      - ./customised_templates/email_confirmation_subscription.html.tmpl:/srv/email_confirmation_subscription.html.tmpl:ro
      - ./customised_templates/email_reply.html.tmpl:/srv/email_reply.html.tmpl:ro
      - ./customised_templates/email_unsubscribe.html.tmpl:/srv/email_unsubscribe.html.tmpl:ro
      - ./customised_templates/error_response.html.tmpl:/srv/error_response.html.tmpl:ro
```

The easiest way to test it is to mount `error_response.html.tmpl`, and then head to <http://127.0.0.1:8080/email/unsubscribe.html>, where you are supposed to see the page like the following:

![Error_template](images/error_template.png)

If the file is mounted correctly, the page will render the new file content immediately after hitting the refresh button in your browser once you change the file.
