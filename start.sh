#!/bin/sh

echo "prepare environment"

# replace base url by REMARK_URL
sed -i "s|BASE_URL:\"[^\"]*\"|BASE_URL:\"${REMARK_URL}\"|g" /srv/web/*.js
sed -i "s|var baseurl = '[^']*';|var baseurl = '${REMARK_URL}';|g" /srv/web/*.html

echo "start remark42 server"

if [ -z "$USER" ] ; then \
    echo "No USER defined, runs under root!"
    exec /srv/remark

else
    echo "runs under ${USER}"
    /sbin/su-exec ${USER} /srv/remark
fi
