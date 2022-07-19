FROM umputun/baseimage:buildgo-v1.9.2 as build-backend

ARG CI
ARG GITHUB_REF
ARG GITHUB_SHA
ARG GIT_BRANCH
ARG SKIP_BACKEND_TEST
ARG BACKEND_TEST_TIMEOUT

ADD backend /build/backend
WORKDIR /build/backend

ENV GOFLAGS="-mod=vendor"

# install gcc in order to be able to go test package with -race
RUN apk --no-cache add gcc libc-dev

RUN echo go version: `go version`

# run tests
RUN \
    cd app && \
    if [ -z "$SKIP_BACKEND_TEST" ] ; then \
        CGO_ENABLED=1 go test -race -p 1 -timeout="${BACKEND_TEST_TIMEOUT:-300s}" -covermode=atomic -coverprofile=/profile.cov_tmp ./... && \
        cat /profile.cov_tmp | grep -v "_mock.go" > /profile.cov ; \
        golangci-lint run --config ../.golangci.yml ./... ; \
    else \
        echo "skip backend tests and linter" \
    ; fi

RUN \
    version="$(/script/version.sh)" && \
    echo "version=$version" && \
    go build -o remark42 -ldflags "-X main.revision=${version} -s -w" ./app

FROM --platform=$BUILDPLATFORM node:16.15.1-alpine as build-frontend-deps

ARG CI
ARG SKIP_FRONTEND_BUILD
ARG SKIP_FRONTEND_TEST
ENV HUSKY_SKIP_INSTALL=true

WORKDIR /srv/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml /srv/frontend/
RUN \
   if [[ -z "$SKIP_FRONTEND_BUILD" || -z "$SKIP_FRONTEND_TEST" ]]; then \
     apk add --no-cache --update git && \
     npm i -g pnpm; \
   fi

RUN --mount=type=cache,id=pnpm,target=/root/.pnpm-store/v3 \
   if [[ -z "$SKIP_FRONTEND_BUILD" || -z "$SKIP_FRONTEND_TEST" ]]; then \
     pnpm i; \
   fi

FROM --platform=$BUILDPLATFORM node:16.15.1-alpine as build-frontend

ARG CI
ARG SKIP_FRONTEND_TEST
ARG SKIP_FRONTEND_BUILD
ARG NODE_ENV=production

COPY --from=build-frontend-deps /srv/frontend/node_modules /srv/frontend/node_modules
COPY ./frontend /srv/frontend
WORKDIR /srv/frontend
RUN \
   if [[ -z "$SKIP_FRONTEND_BUILD" || -z "$SKIP_FRONTEND_TEST" ]]; then \
     apk add --no-cache --update git && \
     npm i -g pnpm; \
   fi

RUN \
  if [ -z "$SKIP_FRONTEND_TEST" ]; then \
    pnpm lint test check; \
  else \
    echo 'Skip frontend test'; \
  fi

RUN \
  if [ -z "$SKIP_FRONTEND_BUILD" ]; then \
    pnpm build; \
  else \
    mkdir -p public; \
    echo 'Skip frontend build'; \
  fi

FROM umputun/baseimage:app-v1.9.2

WORKDIR /srv

ADD docker-init.sh /entrypoint.sh
ADD backend/scripts/backup.sh /usr/local/bin/backup
ADD backend/scripts/restore.sh /usr/local/bin/restore
ADD backend/scripts/import.sh /usr/local/bin/import
RUN chmod +x /entrypoint.sh /usr/local/bin/backup /usr/local/bin/restore /usr/local/bin/import

COPY --from=build-backend /build/backend/remark42 /srv/remark42
COPY --from=build-backend /build/backend/templates /srv
COPY --from=build-frontend /srv/frontend/public/ /srv/web
COPY docker-init.sh /srv/init.sh
RUN chown -R app:app /srv
RUN ln -s /srv/remark42 /usr/bin/remark42

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1


RUN chmod +x /srv/init.sh
CMD ["/srv/remark42", "server"]
