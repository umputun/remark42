FROM --platform=$BUILDPLATFORM node:16.15.1-alpine AS frontend-deps

ARG SKIP_FRONTEND_TEST
ARG SKIP_FRONTEND_BUILD

WORKDIR /srv/frontend/

COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./frontend/pnpm-workspace.yaml /srv/frontend/
COPY ./frontend/apps/remark42/package.json /srv/frontend/apps/remark42/

RUN \
  if [[ -z "$SKIP_FRONTEND_BUILD" || -z "$SKIP_FRONTEND_TEST" ]]; then \
    apk add --no-cache --update git && \
    npm i -g pnpm; \
  fi

RUN --mount=type=cache,id=pnpm,target=/root/.pnpm-store/v3 \
  if [[ -z "$SKIP_FRONTEND_BUILD" || -z "$SKIP_FRONTEND_TEST" ]]; then \
    pnpm i; \
  fi


FROM --platform=$BUILDPLATFORM frontend-deps AS build-frontend

ARG SKIP_FRONTEND_TEST
ARG SKIP_FRONTEND_BUILD
ENV CI=true

WORKDIR /srv/frontend/apps/remark42/

COPY ./frontend/apps/remark42/ /srv/frontend/apps/remark42/

RUN \
  if [ -z "$SKIP_FRONTEND_TEST" ]; then \
    pnpm lint type-check translation-check test; \
  else \
    echo 'Skip frontend test'; \
  fi

RUN \
  if [ -z "$SKIP_FRONTEND_BUILD" ]; then \
    pnpm build; \
  else \
    mkdir /srv/frontend/apps/remark42/public; \
    echo 'Skip frontend build'; \
  fi

FROM umputun/baseimage:buildgo-v1.9.2 as build-backend

ARG CI
ARG GITHUB_REF
ARG GITHUB_SHA
ARG GIT_BRANCH
ARG SKIP_BACKEND_TEST
ARG BACKEND_TEST_TIMEOUT

ADD backend /build/backend
# to embed the frontend files statically into Remark42 binary
COPY --from=build-frontend /srv/frontend/apps/remark42/public/ /build/backend/app/cmd/web/
RUN find /build/backend/app/cmd/web/ -regex '.*\.\(html\|js\|mjs\)$' -print -exec sed -i "s|{% REMARK_URL %}|http://127.0.0.1:8080|g" {} \;
WORKDIR /build/backend

# install gcc in order to be able to go test package with -race
RUN apk --no-cache add gcc libc-dev

RUN echo go version: `go version`

# run tests
RUN \
    cd app && \
    if [ -z "$SKIP_BACKEND_TEST" ] ; then \
        CGO_ENABLED=1 go test -race -p 1 -timeout="${BACKEND_TEST_TIMEOUT:-300s}" -covermode=atomic -coverprofile=/profile.cov_tmp ./... && \
        cat /profile.cov_tmp | grep -v "_mock.go" > /profile.cov && \
        golangci-lint run --config ../.golangci.yml ./... ; \
    else \
      echo "skip backend tests and linter" \
    ; fi

RUN \
    version="$(/script/version.sh)" && \
    echo "version=$version" && \
    go build -o remark42 -ldflags "-X main.revision=${version} -s -w" ./app

FROM umputun/baseimage:app-v1.9.2

ARG GITHUB_SHA

LABEL org.opencontainers.image.authors="Umputun <umputun@gmail.com>" \
      org.opencontainers.image.description="Remark42 comment engine" \
      org.opencontainers.image.documentation="https://remark42.com/docs/getting-started/" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/umputun/remark42.git" \
      org.opencontainers.image.title="Remark42" \
      org.opencontainers.image.url="https://remark42.com/" \
      org.opencontainers.image.revision="${GITHUB_SHA}"

WORKDIR /srv

COPY docker-init.sh /srv/init.sh
ADD backend/scripts/backup.sh /usr/local/bin/backup
ADD backend/scripts/restore.sh /usr/local/bin/restore
ADD backend/scripts/import.sh /usr/local/bin/import
RUN chmod +x /srv/init.sh /usr/local/bin/backup /usr/local/bin/restore /usr/local/bin/import

COPY --from=build-backend /build/backend/remark42 /srv/remark42
COPY --from=build-frontend /srv/frontend/apps/remark42/public/ /srv/web/
RUN chown -R app:app /srv
RUN ln -s /srv/remark42 /usr/bin/remark42

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1

CMD ["/srv/remark42", "server"]
