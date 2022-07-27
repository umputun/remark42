#!/bin/sh
echo "prepare environment"
# replace {% REMARK_URL %} by content of REMARK_URL variable
find . -regex '.*\.\(html\|js\|mjs\)$' -print -exec sed -i "s|{% REMARK_URL %}|${REMARK_URL}|g" {} \;

# replace 'site_id: "remark"' by the first SITE from the comma-separated list of IDs, if present
if [ -n "${SITE}" ]; then
	sep=','
	case ${SITE} in
	*"$sep"*)
		export single_site_id=${SITE%%"$sep"*}
		echo "multiple site IDs passed in SITE (\"${SITE}\"): using \"${single_site_id}\" in frontend site_id"
		;;
	*)
		echo "using non-standard frontend site_id from SITE variable (\"${SITE}\") instead of \"remark\""
		export single_site_id=$SITE
		;;
	esac
	echo "single_site_id: ${single_site_id}"
	sed -i "s|site_id:\"[^\"]*\"|site_id:\"${single_site_id}\"|g" /srv/web/*.html
fi

if [ -d "/srv/var" ]; then
	echo "changing ownership of /srv/var to app:app (remark42 user inside the container)"
	chown -R app:app /srv/var || echo "WARNING: /srv/var ownership change failed, if application will fail that might be the reason"
else
	echo "ERROR: /srv/var doesn't exist, which means that state of the application"
	echo "ERROR: will be lost on container stop or restart."
	echo "ERROR: Please mount local directory to /srv/var in order for it to work."
	exit 199
fi
