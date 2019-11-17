## How to configure remark42 without a subdomain

All README examples show configurations with remark42 on its own subdomain, i.e. `https://remark42.example.com`. However, it is possible and sometimes desirable to run remark42 without a subdomain, but just under some path, i.e.  `https://example.com/remark42`.

- The nginx.conf would then look something like:
```
  location /remark42/ {
    rewrite /remark42/(.*) /$1 break;
    proxy_pass http://remark42:8080/; // use internal docker name of remark42 container for proxy
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
```

- The frontend URL looks like this: `s.src = 'https://example.com/remark42/web/embed.js;`

- The backend `REMARK_URL` parameter will be `https://example.com/remark42`

- And you also need to slightly modify the callback URL for the social media login API's:
  - Facebook Valid OAuth Redirect URIs: `https://example.com/remark42/auth/facebook/callback`
  - Google Authorized redirect URIs: `https://example.com/remark42/auth/google/callback`
  - Github Authorised callback URL: `https://example.com/remark42/auth/github/callback`
  