---
title: Site URL migration
---

Here is an example of how to move your comments after your posts are moved, from `https://example.org/blog/<slug>` to `https://example.org/post/<slug>` in that example.

### Rules file

First, you must create a `rules` file in the remark's `/var` folder.

Format is simply `old_url new_url` like following:

```
https://example.org/old-url-1/ https://example.org/new-url-1/
https://example.org/old-url-2/ https://example.org/new-url-2/
```

### Applying the remap

After rules file is ready, run the following command (`ADMIN_PASSWD` must to be enabled on server for it to work):

```shell
remark42 remap --admin-passwd <password> -s <your site ID> -f var/rules
```

If running in a docker container, the command becomes (`ADMIN_PASSWD` will be taken from the environment):

```shell
docker exec -it remark42 remap -s <your site ID> -f var/rules
```

The command above sends a *request* to start the remap job. You can see the job execution logs by running:

```shell
docker logs <container>
```
