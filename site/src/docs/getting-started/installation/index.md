---
title: Installation
---

## Setup Remark42 Instance on Your Server

### Installation in Docker

_This is the recommended way to run Remark42_

- copy provided [`docker-compose.yml`](https://github.com/umputun/remark42/blob/master/docker-compose.yml) and customize for your needs
- make sure you **don't keep** `ADMIN_PASSWD=something...` for any non-development deployments
- pull prepared images from the Docker Hub and start - `docker-compose pull && docker-compose up -d`
- alternatively, compile from the sources - `docker-compose build && docker-compose up -d`

### Installation with Binary

- download [archive for the stable release](https://github.com/umputun/remark42/releases)
- unpack with `gunzip` (Linux, macOS) or with `zip` (Windows)
- run as `remark42.{os}-{arch} server {parameters...}`, i.e., `remark42.linux-amd64 server --secret=12345 --url=http://127.0.0.1:8080`
- alternatively compile from the sources - `make OS=[linux|darwin|windows] ARCH=[amd64,386,arm64,arm]`

## Setup on Your Website

Add config for Remark on a page of your site ([here](/docs/configuration/frontend/) is the full reference):

- `REMARK_URL` – the URL where is Remark42 instance is served, passed as `REMARK_URL` to backend
- `YOUR_SITE_ID` - the `SITE` that you passed to Remark42 instance on start, `remark` by default.

```html
<script>
  var remark_config = {
    host: 'REMARK_URL',
    site_id: 'YOUR_SITE_ID',
  }
</script>
```

For example:

```html
<script>
  var remark_config = {
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
**Note:** You can place the config with the snippet in any place of the HTML code of your site. If it is closer to start of the HTML (for example in `<head>`) it will start loading sooner and show comments faster.
:::

Put the next code snippet on a page of your site where you want to have comments:

```html
<div id="remark42"></div>
```

After that widget will be rendered inside this node.

For more information about frontend configuration please [learn about other parameters here](https://remark42.com/docs/configuration/frontend/)
If you want to set this up on a Single Page App, see the [appropriate doc page](https://remark42.com/docs/configuration/frontend/spa/).

#### Quick installation test

To verify if Remark42 has been properly installed, check a demo page at `${REMARK_URL}/web` URL. Make sure to include `remark` site ID to the `${SITE}` list.

### Build from the source

* to build Docker container - `make docker`. This command will produce container `umputun/remark42`
* to build a single binary for direct execution - `make OS=<linux|windows|darwin> ARCH=<amd64|386>`. This step will produce an executable `remark42` file with everything embedded
