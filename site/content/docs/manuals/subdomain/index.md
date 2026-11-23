---
title: Configure Instance without Subdomain
---

## How to configure remark42 without a subdomain

All documentation examples show configurations with remark42 on its own subdomain, i.e., `https://remark42.example.com`. However, it is possible and sometimes desirable to run remark42 without a subdomain, but just under some path, i.e., `https://example.com/remark42`.

- In the frontend configuration, set `host: 'https://example.com/remark42'` in `remark_config`. The embed snippet loads the widget from that address, here `https://example.com/remark42/web/embed.mjs`.

- The backend `REMARK_URL` parameter will be `https://example.com/remark42`. Set `ALLOWED_HOSTS="'self'"` to allow embedding only from the same origin. Set `AUTH_SAME_SITE=lax`: an OAuth provider sends the reader back to the callback URL below from its own site, a browser does not send a `strict` cookie with that cross-site request, and the sign-in fails with `failed to get token`.

- And you also need to slightly modify the callback URL for the social media login APIs:
  - Facebook Valid OAuth Redirect URIs: `https://example.com/remark42/auth/facebook/callback`
  - Google Authorized redirect URIs: `https://example.com/remark42/auth/google/callback`
  - GitHub Authorised callback URL: `https://example.com/remark42/auth/github/callback`

### Docker Compose configuration

Both Nginx and Caddy configuration below rely on remark42 available on hostname `remark42`, which is achieved by having `container_name: remark42` in Docker Compose configuration file.

Example `docker-compose.yml`, to which you add the `ALLOWED_HOSTS` and `AUTH_SAME_SITE` settings above and at least one sign-in provider; without a provider, readers cannot sign in to comment:

```yaml
services:
  remark42:
    image: ghcr.io/umputun/remark42:latest
    container_name: remark42
    restart: always
    environment:
      - REMARK_URL=https://example.com/remark42
      - SITE=<site_ID>
      - SECRET=<secret>
      - ADMIN_SHARED_ID=<shared_id>
    volumes:
      - ./data:/srv/var
    logging:
      options:
        max-size: "10m"
        max-file: "1"
```

### Nginx configuration

The `nginx.conf` would then look something like:

```nginx
  location /remark42/ {
    rewrite /remark42/(.*) /$1 break;
    proxy_pass http://remark42:8080/; # use internal docker name of remark42 container for proxy
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
```

Because this configuration sets `X-Real-IP` and `X-Forwarded-For`, set [`--trusted-proxy`](../../configuration/parameters/#trusted-proxies-and-client-ip) to the network Nginx connects from (for a same-compose Nginx that is the Docker network, e.g. `172.16.0.0/12`). Without it Remark42 trusts those headers from any client and logs a warning at start, which lets a client spoof its IP and bypass rate limiting and vote de-duplication.

### Caddy configuration

Example of Caddy 2 configuration (`Caddyfile`) running remark42 service on `example.com/remark42/`, proxying
requests under the path /remark42 through to the docker container:

```
example.com {
  root * /srv/www

  log {
    output file /logs/example.com.access.log
  }

  # remark42
  handle_path /remark42* {
    reverse_proxy remark42:8080
  }

  file_server
}
```

If you are using a legacy version (v1) of Caddy, the following configuration would be appropriate:

```
example.com {
	gzip
	tls mail@example.com

	root /srv/www
	log  /logs/example.com.access.log

	# remark42
	proxy /remark42/ http://remark42:8080/ {
		without /remark42
		transparent
	}
}
```
