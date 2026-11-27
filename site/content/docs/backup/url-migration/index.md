---
title: Site URL migration
---

Here is an example of how to move your comments after your posts are moved, from `https://example.org/blog/<slug>` to `https://example.org/post/<slug>` in that example.

Remap exports the site's comments, clears the site and imports them back with the new addresses, so a failure in the middle leaves the site incomplete: make a [backup](https://remark42.com/docs/backup/backup/) first. Afterwards every comment is marked as imported, and the stored hashes of the voters' IP addresses are gone, so the same-IP vote restriction (`VOTES_IP`) starts afresh.

## Rules file

First, create a `rules` file in the server's data directory, `/srv/var` in the container and `./var` on the host.

Each row holds an old and a new address separated by exactly one space. An address ending in `*` is a prefix: the row moves every address that starts with it, keeping the rest of the address. An exact row wins over a prefix row, and a longer prefix over a shorter one:

```
https://example.org/old-url-1/ https://example.org/new-url-1/
https://example.org/blog/* https://example.org/post/*
```

The second row does the move from the example above. A double space, a tab or an empty row between rules fails the whole request with `bad row`.

On Linux, the container makes its own `app` user the owner of `./var` at every start, which can stop a host user other than root from writing the file there. Write it through the container instead:

```shell
docker exec remark42 sh -c 'echo "https://example.org/blog/* https://example.org/post/*" > var/rules'
```

or set `APP_UID` to your host user's ID to keep `./var` writable for you.

## Applying the remap

After rules file is ready, run the following command (`ADMIN_PASSWD` must be enabled on the server for it to work):

```shell
docker exec -it remark42 remark42 remap -s <your site ID> -f var/rules
```

In `docker exec -it remark42 remark42 ...`, the first `remark42` is the container name and the second is the Remark42 binary inside it. The command takes `ADMIN_PASSWD`, `REMARK_URL` and `SECRET` from the container's environment. It sends the request to `REMARK_URL`, the public address, so the request goes through your reverse proxy; to send it to the server directly, add `--url=http://127.0.0.1:8080`, using the port the server listens on inside the container.

Outside the container, set `REMARK_URL` and `SECRET` in the environment, or pass `--url` and `--secret`:

```shell
remark42 remap --url <REMARK_URL> --secret <SECRET> --admin-passwd <password> -s <your site ID> -f var/rules
```

The command above sends a _request_ to start the remap job and exits before the job is done. `docker logs remark42` shows `imported N comments from M records` when it is done, or a line with `import failed`.
