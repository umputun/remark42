---
title: Installation
parent: Getting Started
order: 100
---

## Backend

### Installation with Docker

_This is the recommended way to run remark42_

- copy provided `docker-compose.yml` and customize for your needs
- make sure you **don't keep** `ADMIN_PASSWD=something...` for any non-development deployments
- pull prepared images from the DockerHub and start - `docker-compose pull && docker-compose up -d`
- alternatively compile from the sources - `docker-compose build && docker-compose up -d`

**TODO: add list of necessary parameters**

### Installation with Binary

- download archive for [stable release](https://github.com/umputun/remark42/releases) or [development version](https://remark42.com/downloads)
- unpack with `gunzip` (Linux, macOS) or with `zip` (Windows)
- run as `remark42.{os}-{arch} server {parameters...}`, i.e. `remark42.linux-amd64 server --secret=12345 --url=http://127.0.0.1:8080`
- alternatively compile from the sources - `make OS=[linux|darwin|windows] ARCH=[amd64,386,arm64,arm]`

**TODO: add list of necessary parameters**

## Setup on Your Website

Put the next code snippet in place where you want to have comments:

```
<div id="remark42"></div>
```

Add this snippet to the bottom of web page:

```html
<script>
	var remark_config = {
		host: 'REMARK_URL',
		site_id: 'YOUR_SITE_ID',
	}
</script>
<script>
	!(function (e, n) {
		for (var o = 0; o < e.length; o++) {
			var r = n.createElement('script'),
				c = '.js',
				d = n.head || n.body
			'noModule' in r ? ((r.type = 'module'), (c = '.mjs')) : (r.async = !0),
				(r.defer = !0),
				(r.src = remark_config.host + '/web/' + e[o] + c),
				d.appendChild(r)
		}
	})(remark_config.components || ['embed'], document)
</script>
```

And then add this node in the place where you want to see Remark42 widget:

```html
<div id="remark42"></div>
```

After that widget will be rendered inside this node.

If you want to set this up on a Single Page App, see [appropriate doc page](https://github.com/umputun/remark42/blob/master/docs/latest/spa.md).
