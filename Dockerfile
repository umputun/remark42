FROM umputun/baseimage:buildgo-v1.8.0 as build-backend

ARG CI
ARG GITHUB_REF
ARG GITHUB_SHA
ARG GIT_BRANCH
ARG SKIP_BACKEND_TEST
ARG BACKEND_TEST_TIMEOUT

ADD backend /build/backend
ADD .git/ /build/backend/.git/
WORKDIR /build/backend

ENV GOFLAGS="-mod=vendor"

# install gcc in order to be able to go test package with -race
RUN apk --no-cache add gcc libc-dev

# run tests
RUN \
    cd app && \
    if [ -z "$SKIP_BACKEND_TEST" ] ; then \
        CGO_ENABLED=1 go test -race -p 1 -timeout="${BACKEND_TEST_TIMEOUT:-300s}" -covermode=atomic -coverprofile=/profile.cov_tmp ./... && \
        cat /profile.cov_tmp | grep -v "_mock.go" > /profile.cov ; \
        golangci-lint run --config ../.golangci.yml ./... ; \
    else echo "skip backend tests and linter" ; fi

RUN \
    version="$(/script/version.sh)" && \
    echo "version=$version" && \
    go build -o remark42 -ldflags "-X main.revision=${version} -s -w" ./app

FROM --platform=$BUILDPLATFORM node:16.13.2-alpine as build-frontend-deps

ARG CI
ENV HUSKY_SKIP_INSTALL=true

RUN apk add --no-cache --update git
ADD frontend/package.json /srv/frontend/package.json
ADD frontend/package-lock.json /srv/frontend/package-lock.json
RUN cd /srv/frontend && CI=true npm ci --loglevel warn

FROM --platform=$BUILDPLATFORM node:16.13.2-alpine as build-frontend

ARG CI
ARG SKIP_FRONTEND_TEST
ARG NODE_ENV=production

COPY --from=build-frontend-deps /srv/frontend/node_modules /srv/frontend/node_modules
ADD frontend /srv/frontend
RUN cd /srv/frontend && \
    if [ -z "$SKIP_FRONTEND_TEST" ] ; then npm run lint test check; \
    else echo "skip frontend tests and lint" ; npm run build ; fi && \
    rm -rf ./node_modules

FROM umputun/baseimage:app-v1.8.0

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
