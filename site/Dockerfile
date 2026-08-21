FROM alpine:3.22 AS build

ARG HUGO_VERSION=0.165.0
ARG TARGETARCH

# the extended build is glibc-linked and will not run on alpine; nothing here needs
# sass or webp, so the plain static build is the one to use
RUN apk add --no-cache ca-certificates \
    && case "${TARGETARCH}" in \
         amd64) HUGO_ARCH=amd64 ;; \
         arm64) HUGO_ARCH=arm64 ;; \
         "") echo "TARGETARCH is empty; build with buildkit" >&2; exit 1 ;; \
         *) echo "unsupported arch ${TARGETARCH}" >&2; exit 1 ;; \
       esac \
    && HUGO_TAR="hugo_${HUGO_VERSION}_linux-${HUGO_ARCH}.tar.gz" \
    && BASE="https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}" \
    && wget -qO "/tmp/${HUGO_TAR}" "${BASE}/${HUGO_TAR}" \
    && wget -qO /tmp/checksums.txt "${BASE}/hugo_${HUGO_VERSION}_checksums.txt" \
    && grep " ${HUGO_TAR}$" /tmp/checksums.txt > /tmp/expected.sha256 \
    && test -s /tmp/expected.sha256 \
    && (cd /tmp && sha256sum -c expected.sha256) \
    && tar -xzf "/tmp/${HUGO_TAR}" -C /usr/local/bin hugo \
    && rm -f "/tmp/${HUGO_TAR}" /tmp/checksums.txt /tmp/expected.sha256 \
    && hugo version

WORKDIR /site
COPY ./ /site
RUN hugo --minify --destination /site/build \
    && test -s /site/build/index.html \
    && test -s /site/build/docs/getting-started/installation/index.html

FROM ghcr.io/umputun/reproxy
LABEL org.opencontainers.image.authors="Umputun <umputun@gmail.com>" \
      org.opencontainers.image.description="Remark42 site" \
      org.opencontainers.image.documentation="https://github.com/umputun/remark42/tree/master/site" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/umputun/remark42" \
      org.opencontainers.image.title="Remark42 site" \
      org.opencontainers.image.url="https://remark42.com/"

COPY --from=build /site/build /srv/site
EXPOSE 8080
USER app
ENTRYPOINT ["/srv/reproxy", "--assets.location=/srv/site", "--assets.not-found=404.html"]
