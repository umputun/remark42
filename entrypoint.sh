#!/bin/sh

# replace BASE_URL constant by REMARK_URL
sed -i "s|https://demo.remark42.com|${REMARK_URL}|g" /srv/web/*.js
# remove devtools attach helper. TODO: move to webpack loader
sed -i "/REMOVE-START/,/REMOVE-END/d" /srv/web/iframe.html

if [ -z "$USER" ] ; then \
    echo "No USER defined, runs under root!"
    exec /srv/remark42 $@

else
    /sbin/su-exec ${USER} /srv/remark42 $@
fi
