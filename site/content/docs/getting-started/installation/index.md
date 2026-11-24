---
title: Installation
aliases:
  - /docs/getting-started/
  - /docs/
  - /docs/getting-started/system-requirements/
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
- pull prepared images from the GitHub Container Registry and start - `docker compose pull && docker compose up -d`
- alternatively, compile from the sources - `docker compose build && docker compose up -d`

### Installation with Binary

- download [archive for the stable release](https://github.com/umputun/remark42/releases)
- extract the `.tar.gz` archive with `tar -xzf <archive>.tar.gz` (Linux, macOS, FreeBSD), or extract the `.zip` archive (Windows)
- run as `remark42.{os}-{arch} server {parameters...}`, i.e., `remark42.linux-amd64 server --secret=12345 --url=http://127.0.0.1:8080`
- alternatively, [build from the source](#build-from-the-source)

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
   # directory where data files are stored and automatic backups are created
   WorkingDirectory=/var/www/remark42
   Restart=on-failure
   # another good alternative is `www-data`
   User=nobody
   # another good alternative is `www-data`
   Group=nogroup

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

### Build from the source

- to build a Docker image, run `make docker`. It produces an image tagged both `umputun/remark42` and `ghcr.io/umputun/remark42`
- to build a single binary for direct execution, run `make OS=<linux|darwin|freebsd|windows> ARCH=<amd64|arm64|386>` (`linux` and `amd64` by default). This requires Go 1.27, Node 24+, PNPM 10 and Git, builds the frontend assets locally and produces an executable `remark42` file with everything embedded

## Setup on Your Website

Add config for Remark on a page of your site ([here](/docs/configuration/frontend/) is the full reference):

- `REMARK_URL` – the URL where the Remark42 instance is served, passed as `REMARK_URL` to backend
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
<script>!function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),d=n.head||n.body;r.type="module",r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+".mjs",d.appendChild(r)}}(remark_config.components||["embed"],document);</script>
```
<!-- prettier-ignore-end -->

{{< note "💡" >}}
**Note:** You can place the config with the snippet in any place of the HTML code of your site. If it is closer to start of the HTML (for example in `<head>`) it will start loading sooner and show comments faster.
{{< /note >}}

Put the next code snippet on a page of your site where you want to have comments:

```html
<div id="remark42" style="min-height: 600px">Comments loading...</div>
```

After that widget will be rendered inside this node. Any content you place inside the div (such as "Comments loading..." above) is automatically removed once the widget initialises, so you can use it as a loading placeholder. The widget grows to the height of the comments once they load, which moves the content below it; the `min-height` reserves space so that the content moves less. Set it close to the usual height of your comments, or leave it out.

For more information about frontend configuration please [learn about other parameters here](/docs/configuration/frontend/).
If you want to set this up on a Single Page App, see the [appropriate doc page](/docs/configuration/frontend/spa/).

### Quick installation test

To verify if Remark42 has been properly installed, check a demo page at `${REMARK_URL}/web` URL. It shows the comments of the first site ID listed in `SITE`.

## Privacy

Remark42 stores personal data about commenters, and whoever runs an instance decides what happens to it. This section lists the settings and procedures that affect that data; it is not legal advice.

- **Replace the built-in privacy policy.** The page served at `/web/privacy.html` is a default text written for remark42.com and does not describe your site. To serve your own, add a `privacy.html` to the frontend files in the `web-root` / `REMARK_WEB_ROOT` directory (`/srv/web` in the Docker image); a file there is served in preference to the built-in one. Use a directory that holds the full frontend build: an existing web root replaces the embedded frontend, so a directory containing only `privacy.html` cannot serve the widget.
- **Turn on the image proxy.** Setting `IMAGE_PROXY_CACHE_EXTERNAL=true` is strongly recommended; it is off by default. Without it, pictures that commenters embed from other sites can load directly in readers' browsers, which shows each reader's IP address to the picture's host. With it on, Remark42 fetches and caches such pictures on your instance, and the widget's Content-Security-Policy allows pictures from your instance only. Existing comments are not rewritten: pictures whose stored addresses still point to other hosts stop loading, and pictures already served through your instance keep working.
- **Know that uploaded pictures keep their metadata.** A picture a commenter uploads is stored and served as the file was sent when it fits within the resize limits (`IMAGE_RESIZE_WIDTH` by `IMAGE_RESIZE_HEIGHT`, 2400 by 900 pixels by default). That includes the metadata in the file, such as the place and time a photo was taken and the camera it was taken with. A larger picture is scaled down and saved as a new PNG file without that metadata. With either limit set to `0` no picture is scaled, and every picture is kept as sent. Remark42 does not remove metadata from a picture it keeps as sent.
- **Know what leaves your server.** Apart from the sign-in provider a user chooses, Remark42 sends personal data to other services in these cases:
  - With email sign-in enabled (`AUTH_EMAIL_ENABLE`), each sign-in makes the server ask Gravatar for a picture, sending an MD5 hash of the address.
  - Sign-in codes and email notifications go to the recipient's address through the SMTP server you configure.
  - Notifications (`NOTIFY_USERS`, `NOTIFY_ADMINS`) send the text of a comment and, depending on the channel, the commenter's name, the page address and title and the comment it replies to. They go to email, Telegram, Slack or your webhook.
- **Handle deletion requests.** A signed-in user has a "Request deletion of my data" button in their profile. The button deletes nothing and the server sends no email: it opens the user's mail client with a prepared message to the administrator's address (`ADMIN_SHARED_EMAIL`, by default `admin@` plus the host of `REMARK_URL`), so make sure that mailbox is read. The message carries a link that is valid for three months. When an administrator of that site opens it while signed in, the text of the user's comments, their name and ID on those comments, the pictures they uploaded that those comments still show and their notification addresses are removed on that site, and their avatar is removed for the whole instance, since avatars are not kept per site. A picture that another user's comment on the same page also shows is kept. The votes the user gave on other comments stay, stored under the user ID, and so do the user's entries in the block and verified lists. Anonymous users have no such button; their data can be deleted only by an administrator calling [`DELETE /api/v1/admin/user/{userid}`](/docs/contributing/api/#admin).
- **Remember that backups keep deleted data.** [Automatic backups](/docs/backup/backup/) keep comments and notification addresses that were deleted after the backup was made. Remark42 keeps up to `MAX_BACKUP_FILES` files per site (10 by default) and rotates the files in `BACKUP_PATH` named `backup-{site ID}-{YYYYMMDD}.gz`; manual backups and copies elsewhere are yours to manage. A [restore](/docs/backup/restore/) brings back the deleted comments and notification addresses held in that backup, so repeat those deletions afterwards. Backups do not include avatar and picture files.
- **Decide how long logs are kept.** Remark42 writes its log to standard output and standard error and has no retention setting of its own, so the log lives as long as your container runtime or service manager keeps it. It holds personal data: API and sign-in requests are logged with their URL and query, the beginning of the request body, such as the text of a comment, and a hash of the client's IP address, and some error lines, including sign-in, avatar and failed picture or RSS responses, carry the IP address itself. The request and sign-in lines mask email addresses and JWTs; other token formats are not masked. With `DEBUG=true`, which the provided `docker-compose.yml` carries as a commented line, the log also records each new comment in full with its author's name and user ID, the name and user ID of a user who signs in through a social login, and those of any signed-in user whenever a request renews the session; turn it on only while you need debug output. The provided `docker-compose.yml` limits the log by size, not by age. Set retention separately for the log of any reverse proxy in front of Remark42.
