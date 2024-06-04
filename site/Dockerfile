FROM node:20-alpine as build

WORKDIR /site
COPY ./ /site
RUN yarn --frozen-lockfile
RUN yarn build
RUN ls -la /site

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
ENTRYPOINT ["/srv/reproxy", "--assets.location=/srv/site"]
