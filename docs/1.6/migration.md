---
title: Site URL migration
---

Here is an example on how to move your comments after your posts are moved, from `https://example.org/blog/<slug>` to `https://example.org/post/<slug>` in that example.

### Rules file

First you have to create a `rules` file in remark's `/var` folder.

[Here is the test](https://github.com/akosourov/remark/blob/4dc123dbe84f4f248864bcdbbd6cc2b3a4dafe11/backend/app/migrator/mapper_test.go#L13-L17) with an example of rules file syntax, format is simply `old_url new_url` like following:

```
https://example.org/old-url-1/ https://example.org/new-url-1/
https://example.org/old-url-2/ https://example.org/new-url-2/
```

### Applying the remap

After rules file is ready, run the following command:

```sh
remark42 remap --admin-passwd <password> -f var/rules
```

If running in a docker container, the command becomes:
```sh
docker ps # to find the container name
docker exec -it <container> remark42 remap --admin-passwd <password> -f var/rules
```
