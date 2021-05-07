FROM umputun/baseimage:buildgo-latest as build-backend

ADD backend /build/backend
WORKDIR /build/backend/_example/memory_store

RUN go build -o /build/bin/memory_store -ldflags "-X main.revision=0.0.0 -s -w"


FROM umputun/baseimage:app-latest

WORKDIR /srv
COPY --from=build-backend /build/bin/memory_store /srv/memory_store
RUN chown -R app:app /srv

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1
USER app

CMD ["/srv/memory_store"]
