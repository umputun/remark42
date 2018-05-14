#!/bin/sh
echo "import disqus file $1 to site $2"
curl -X POST -H "Content-Type: application/json" -d @/srv/var/$1 "http://127.0.0.1:8081/api/v1/admin/import?site=${2}&provider=disqus&secret=${SECRET}"
