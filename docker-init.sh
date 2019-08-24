#!/bin/sh
echo "prepare environment"
# replace BASE_URL constant by REMARK_URL
sed -i "s|https://demo.remark42.com|${REMARK_URL}|g" /srv/web/*.js
# remove devtools attach helper. TODO: move to webpack loader
sed -i "/REMOVE-START/,/REMOVE-END/d" /srv/web/iframe.html

chown -R app:app /srv/var 2>/dev/null
