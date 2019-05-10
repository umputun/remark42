#!/bin/sh
chown -R app:app /srv/var 2>/dev/null

echo "start remark42 server"
/sbin/su-exec app /srv/remark42 $@
