#!/bin/sh

echo "prepare environment"

# replace BASE_URL constant by REMARK_URL
sed -i "s|https://demo.remark42.com|${REMARK_URL}|g" /srv/web/*.js
# remove devtools attach helper. TODO: move to webpack loader
sed -i "/REMOVE-START/,/REMOVE-END/d" /srv/web/iframe.html

echo "start remark42 server"

if [ -z "$USER" ] ; then \
    echo "No USER defined, runs under root!"
    exec /srv/remark

else
    echo "runs under ${USER}"
    /sbin/su-exec ${USER} /srv/remark
fi
