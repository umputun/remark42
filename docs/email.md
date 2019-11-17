## Setup email authentication

To allow email auth `AUTH_EMAIL_ENABLE` should be set to `true`. In addition, user needs to configure SMTP details with

```
AUTH_EMAIL_HOST         
AUTH_EMAIL_PORT         
AUTH_EMAIL_FROM         
AUTH_EMAIL_SUBJ         
AUTH_EMAIL_CONTENT_TYPE 
AUTH_EMAIL_TLS          
AUTH_EMAIL_USER         
AUTH_EMAIL_PASSWD       
AUTH_EMAIL_TIMEOUT      
AUTH_EMAIL_TEMPLATE
```

This is an example of configuration using mailgun email service:

```
      - AUTH_EMAIL_ENABLE=true
      - AUTH_EMAIL_HOST=smtp.mailgun.org
      - AUTH_EMAIL_PORT=465
      - AUTH_EMAIL_TLS=true
      - AUTH_EMAIL_USER=postmaster@mg.example.com
      - AUTH_EMAIL_PASSWD=*********
      - AUTH_EMAIL_FROM=confirmation@example.com
```

Configuration example for gmail:

```
      - AUTH_EMAIL_ENABLE=true
      - AUTH_EMAIL_HOST=smtp.gmail.com
      - AUTH_EMAIL_PORT=465
      - AUTH_EMAIL_FROM=example.user@gmail.com
      - AUTH_EMAIL_SUBJ=Comments email confirmation
      - AUTH_EMAIL_TLS=true
      - AUTH_EMAIL_USER=example.user@gmail.com
      - AUTH_EMAIL_PASSWD=secretpassword
```

Usually you don't need to change/set anything else. In case if you want to use a different email template set `AUTH_EMAIL_TEMPLATE`, for instance
`- AUTH_EMAIL_TEMPLATE="Confirmation email, token: {{.Token}}"`. See [verified-authentication](https://github.com/go-pkgz/auth#verified-authentication) for more details.

 