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

After rules file is ready, run the following command:

```shell
remark42 remap --admin-passwd <password> -f var/rules
```

If running in a docker container, the command becomes:

```shell
docker ps # to find the container name
docker exec -it <container> remark42 remap --admin-passwd <password> -f var/rules
```
