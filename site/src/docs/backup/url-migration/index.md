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

#### Tips

1. The command above sends a *request* to start the remap job. You can see the job execution logs by running:

```shell
docker logs <container>
```

2. If you see in logs an entry similar to `export failed with site "site1.com,site2" not found`, please run the command again and specify desired site with command line arguments. For example:

```shell
docker exec -it <container> remark42 remap --admin-passwd <password> -f var/rules --site site1.com
``` 


