#!/bin/sh

echo "prepare environment"

# replace base url by REMARK_URL
sed -i "s|BASE_URL:\"[^\"]*\"|BASE_URL:\"${REMARK_URL}\"|g" /srv/web/*.js
sed -i "s|var baseurl = '[^']*';|var baseurl = '${REMARK_URL}';|g" /srv/web/*.html

echo "start remark42 server"

/sbin/su-exec ${USER} /srv/remark server
