#!/bin/sh
# Self-signed certificate for the https services in the e2e stack.
#
# The suite passes IgnoreHTTPSErrors, so the certificate is never validated and its contents do
# not have to satisfy anything. It carries the right names anyway, which removes the name mismatch
# from the list of complaints when somebody looks at the stack by hand. The issuer is still unknown,
# so curl needs -k and a browser needs an exception either way.
#
# Regenerated only when missing: a fresh certificate on every stack-up would give the running
# services one the containers were not started with, since nginx and remark42 read it once.
set -eu

cd "$(dirname "$0")"

if [ -f cert.pem ] && [ -f key.pem ]; then
	exit 0
fi

# stdout only. openssl writes its progress to stderr and its errors there too, and dropping both
# leaves every caller with a bare exit code: the CI step, `make e2e-up` and ensureStack all print
# nothing about why the certificate could not be made
openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
	-keyout key.pem -out cert.pem \
	-subj "/CN=remark42-e2e" \
	-addext "subjectAltName=DNS:remark42-https,DNS:host-site-https,DNS:localhost,IP:127.0.0.1" \
	>/dev/null

chmod 644 key.pem cert.pem
