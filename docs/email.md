## Overview

This documentation describes how to enable email-related capabilities of Remark. Currently following is possible:

- email authentication for users:

    enabling this will let user log in using their emails;

    ![Email authentication](/docs/images/email_auth.png?raw=true)

- email notifications for any users except anonymous:

    GitHub or Google or Twitter or any other kind of user gets ability to get email notifications about new relies to their comments.

    ![Email notifications subscription](/docs/images/email_notifications.png?raw=true)

## Setup email server connection

In order to enable any of email functionality you need to set up email (SMTP) server connection using these variables:

```
SMTP_HOST
SMTP_PORT
SMTP_TLS
SMTP_USERNAME
SMTP_PASSWORD
SMTP_TIMEOUT
```

### Mailgun

This is an example of configuration using [Mailgun](https://www.mailgun.com/) email service:

```
      - SMTP_HOST=smtp.eu.mailgun.org
      - SMTP_PORT=465
      - SMTP_TLS=true
      - SMTP_USERNAME=postmaster@mg.example.com
      - SMTP_PASSWORD=secretpassword
      - AUTH_EMAIL_FROM=notify@example.com
```

### Gmail

Configuration example for Gmail:

```
      - SMTP_HOST=smtp.gmail.com
      - SMTP_PORT=465
      - SMTP_TLS=true
      - SMTP_USERNAME=example.user@gmail.com
      - SMTP_PASSWORD=secretpassword
      - AUTH_EMAIL_FROM=example.user@gmail.com
```


## Setup email authentication

Here is the list of variables which affect email authentication:

```
AUTH_EMAIL_ENABLE
AUTH_EMAIL_FROM
AUTH_EMAIL_SUBJ
AUTH_EMAIL_CONTENT_TYPE
AUTH_EMAIL_TEMPLATE
```

After `SMTP_` variables are set, you can allow email authentication by setting these two variables:

```
      - AUTH_EMAIL_ENABLE=true
      - AUTH_EMAIL_FROM=notify@example.com
```


Usually you don't need to change/set anything else. In case if you want to use a different email template set `AUTH_EMAIL_TEMPLATE`, for instance
`- AUTH_EMAIL_TEMPLATE="Confirmation email, token: {{.Token}}"`. See [verified-authentication](https://github.com/go-pkgz/auth#verified-authentication) for more details.

## Setup email notifications

Here is the list of variables which affect email notifications:

```
NOTIFY_TYPE
NOTIFY_EMAIL_FROM
NOTIFY_EMAIL_VERIFICATION_SUBJ
```

After `SMTP_` variables are set, you can allow email notifications by setting these two variables:

```
      - NOTIFY_TYPE=email
      # - NOTIFY_TYPE=email,telegram # this is in case you want to have both email and telegram notifications enabled
      - NOTIFY_EMAIL_FROM=notify@example.com
```
