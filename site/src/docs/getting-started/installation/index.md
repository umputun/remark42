---
title: Installation
parent: Getting Started
order: 100
---

## Setup Remark42 Instance on Your Server

### Installation in Docker

_This is the recommended way to run remark42_

- copy provided `docker-compose.yml` and customize for your needs
- make sure you **don't keep** `ADMIN_PASSWD=something...` for any non-development deployments
- pull prepared images from the DockerHub and start - `docker-compose pull && docker-compose up -d`
- alternatively compile from the sources - `docker-compose build && docker-compose up -d`

### Installation with Binary

- download archive for [stable release](https://github.com/umputun/remark42/releases) or [development version](https://remark42.com/downloads)
- unpack with `gunzip` (Linux, macOS) or with `zip` (Windows)
- run as `remark42.{os}-{arch} server {parameters...}`, i.e. `remark42.linux-amd64 server --secret=12345 --url=http://127.0.0.1:8080`
- alternatively compile from the sources - `make OS=[linux|darwin|windows] ARCH=[amd64,386,arm64,arm]`

## Setup on Your Website

Add config for Remark on a page of your site:

```html
<script>
	const remark_config = {
		host: 'REMARK_URL',
		site_id: 'YOUR_SITE_ID',
	}
</script>
```

- `REMARK_URL` – the URL where is Remark42 instance is served.
- `YOUR_SITE_ID` - the ID that you passed to Remark42 instance on start.

For example:

```html
<script>
	const remark_config = {
		host: 'https://demo.remark42.com',
		site_id: 'remark',
	}
</script>
```

After that place the code snippet right after config.

<!-- prettier-ignore-start -->
```html
<script>!function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),c=".js",d=n.head||n.body;"noModule"in r?(r.type="module",c=".mjs"):r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+c,d.appendChild(r)}}(remark_config.components||["embed"],document);</script>
```
<!-- prettier-ignore-end -->

::: note 💡
**Note that:** You can place the config with the snippet in any place of the HTML code of your site. If it closer to start of the HTML (for example in `<head>`) it will start loading sooner and show comments faster.
:::

Put the next code snippet on a page of your site where you want to have comments:

```html
<div id="remark42"></div>
```

After that widget will be rendered inside this node.

If you want to set this up on a Single Page App, see [appropriate doc page](https://github.com/umputun/remark42/blob/master/docs/latest/spa.md).
