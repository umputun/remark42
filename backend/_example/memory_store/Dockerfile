FROM umputun/baseimage:buildgo-v1.13.0 as build-backend

ADD backend /build/backend
WORKDIR /build/backend/_example/memory_store

RUN go build -o /build/bin/memory_store -ldflags "-X main.revision=0.0.0 -s -w"

FROM umputun/baseimage:app-v1.13.0

ARG GITHUB_SHA

LABEL org.opencontainers.image.authors="Umputun <umputun@gmail.com>" \
      org.opencontainers.image.description="Remark42 comment engine example JRPC memory store" \
      org.opencontainers.image.documentation="https://github.com/umputun/remark42/tree/master/backend/_example/memory_store" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/umputun/remark42" \
      org.opencontainers.image.title="Remark42 JRPC example memory store" \
      org.opencontainers.image.url="https://remark42.com/" \
      org.opencontainers.image.revision="${GITHUB_SHA}"

WORKDIR /srv
COPY --from=build-backend /build/bin/memory_store /srv/memory_store
RUN chown -R app:app /srv

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1
USER app

CMD ["/srv/memory_store"]
