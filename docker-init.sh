#!/bin/sh
echo "prepare environment"
# replace BASE_URL constant by REMARK_URL
sed -i "s|https://demo.remark42.com|${REMARK_URL}|g" /srv/web/*.js
# remove devtools attach helper. TODO: move to webpack loader
sed -i "/REMOVE-START/,/REMOVE-END/d" /srv/web/iframe.html

if [ -d "/srv/var" ]; then
  chown -R app:app /srv/var 2>/dev/null
else
  echo "ERROR: /srv/var doesn't exist, which means that state of the application"
  echo "ERROR: will be lost on container stop or restart."
  echo "ERROR: Please mount local directory to /srv/var in order for it to work."
  exit 199
fi
