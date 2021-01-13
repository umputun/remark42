#!/bin/sh
echo "prepare environment"
# replace {% REMARK_URL %} by content of REMARK_URL variable
find . -regex '.*\.\(html\|js\|mjs\)$' -print -exec sed -i "s|{% REMARK_URL %}|${REMARK_URL}|g" {} \;

if [ -n "${SITE_ID}" ]; then
  #replace "site_id: 'remark'" by SITE_ID
  sed -i "s|'remark'|'${SITE_ID}'|g" /srv/web/*.html
fi

if [ -d "/srv/var" ]; then
  chown -R app:app /srv/var 2>/dev/null
else
  echo "ERROR: /srv/var doesn't exist, which means that state of the application"
  echo "ERROR: will be lost on container stop or restart."
  echo "ERROR: Please mount local directory to /srv/var in order for it to work."
  exit 199
fi
