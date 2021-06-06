---
title: Subdomain
---

## How to configure remark42 without a subdomain

All README examples show configurations with remark42 on its own subdomain, i.e. `https://remark42.example.com`. However, it is possible and sometimes desirable to run remark42 without a subdomain, but just under some path, i.e. `https://example.com/remark42`.

- The frontend URL looks like this: `s.src = 'https://example.com/remark42/web/embed.js;`

- The backend `REMARK_URL` parameter will be `https://example.com/remark42`

- And you also need to slightly modify the callback URL for the social media login API's:
  - Facebook Valid OAuth Redirect URIs: `https://example.com/remark42/auth/facebook/callback`
  - Google Authorized redirect URIs: `https://example.com/remark42/auth/google/callback`
  - GitHub Authorised callback URL: `https://example.com/remark42/auth/github/callback`

### docker-compose configuration

Both Nginx and Caddy configuration below relies on remark42 available on hostname `remark42`, which is achieved by having `container_name: remark42` in docker-compose.

Example `docker-compose.yaml`:

```yaml
version: '2'
services:
  remark42:
    image: umputun/remark42:latest
    container_name: remark42
    restart: always
    environment:
      - REMARK_URL=https://example.com/remark42/
      - SITE=<site_ID>
      - SECRET=<secret>
      - ADMIN_SHARED_ID=<shared_id>
    volumes:
      - ./data:/srv/var
    logging:
      options:
        max-size: '10m'
        max-file: '1'
```

### Nginx configuration

The `nginx.conf` would then look something like:

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

### Caddy configuration

Example of Caddy configuration (`Caddyfile`) running remark42 service on `example.com/remark42/`:

```caddy
example.com {
	gzip
	tls mail@example.com

	root /srv/www
	log  /logs/access.log

	# remark42
	proxy /remark42/ http://remark42:8080/ {
		without /remark42
		transparent
	}
}
```
