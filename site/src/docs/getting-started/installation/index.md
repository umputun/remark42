---
title: Installation
---

## System Requirements

Remark42 is designed to be lightweight and efficient. Based on production usage statistics from busy installations:

- **CPU**: Minimal usage (typically under 0.1%)
- **Memory**: ~80MiB RAM (less than 4% of 2GB)
- **Network**: Moderate bandwidth requirements
- **Disk**: Small footprint (under 200MB for a 5-year-old installation with regular activity)

These requirements make Remark42 suitable for running on small VPS instances, shared hosting environments, or even on Raspberry Pi and similar devices.

## Setup Remark42 Instance on Your Server

### Installation in Docker

_This is the recommended way to run Remark42_

- copy provided [`docker-compose.yml`](https://github.com/umputun/remark42/blob/master/docker-compose.yml) and customize for your needs
- make sure you **don't keep** `ADMIN_PASSWD=something...` for any non-development deployments
- pull prepared images from the Docker Hub and start - `docker compose pull && docker compose up -d`
- alternatively, compile from the sources - `docker compose build && docker compose up -d`

### Installation with Binary

- download [archive for the stable release](https://github.com/umputun/remark42/releases)
- unpack with `gunzip` (Linux, macOS) or with `zip` (Windows)
- run as `remark42.{os}-{arch} server {parameters...}`, i.e., `remark42.linux-amd64 server --secret=12345 --url=http://127.0.0.1:8080`
- alternatively compile from the sources - `make OS=[linux|darwin|windows] ARCH=[amd64,386,arm64,arm]`

#### Installation as a systemd Service

For a clean persistent setup without lengthy command line parameters:

1. Create an environment file `/etc/remark42.env`:
   ```
   SECRET=12345
   REMARK_URL=http://127.0.0.1:8080
   ```

1. Create a systemd service file `/etc/systemd/system/remark42.service`:
   ```
   [Unit]
   Description=Remark42 Commenting Server
   After=syslog.target
   After=network.target

   [Service]
   Type=simple
   EnvironmentFile=/etc/remark42.env
   ExecStart=/usr/local/bin/remark42 server
   WorkingDirectory=/var/www/remark42       # directory where data files are stored and automatic backups will be created
   Restart=on-failure
   User=nobody                              # another good alternative is `www-data`
   Group=nogroup                            # another good alternative is `www-data`

   [Install]
   WantedBy=multi-user.target
   ```

1. Enable and start the service:
   ```
   sudo systemctl enable remark42.service
   sudo systemctl start remark42.service
   ```

1. To update configuration, edit the environment file and restart the service:
   ```
   sudo systemctl restart remark42.service
   ```

## Setup on Your Website

Add config for Remark on a page of your site ([here](/docs/configuration/frontend/) is the full reference):

- `REMARK_URL` – the URL where is Remark42 instance is served, passed as `REMARK_URL` to backend
- `YOUR_SITE_ID` - the `SITE` that you passed to Remark42 instance on start, `remark` by default.

```html
<script>
	var remark_config = {
		host: "REMARK_URL",
		site_id: "YOUR_SITE_ID",
	}
</script>
```

For example:

```html
<script>
	var remark_config = {
		host: "https://demo.remark42.com",
		site_id: "remark",
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
<div id="remark42">Comments loading...</div>
```

After that widget will be rendered inside this node. Any content you place inside the div (such as "Comments loading..." above) is automatically removed once the widget initialises, so you can use it as a loading placeholder.

For more information about frontend configuration please [learn about other parameters here](https://remark42.com/docs/configuration/frontend/)
If you want to set this up on a Single Page App, see the [appropriate doc page](https://remark42.com/docs/configuration/frontend/spa/).

#### Quick installation test

To verify if Remark42 has been properly installed, check a demo page at `${REMARK_URL}/web` URL. Make sure to include `remark` site ID to the `${SITE}` list.

### Build from the source

- to build Docker container - `make docker`. This command will produce container `ghcr.io/umputun/remark42`
- to build a single binary for direct execution - `make OS=<linux|windows|darwin> ARCH=<amd64|386>`. This step will produce an executable `remark42` file with everything embedded
