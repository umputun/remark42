FROM umputun/baseimage:buildgo-latest as build-backend

ARG COVERALLS_TOKEN
ARG CI
ARG TRAVIS
ARG TRAVIS_BRANCH
ARG TRAVIS_COMMIT
ARG TRAVIS_JOB_ID
ARG TRAVIS_JOB_NUMBER
ARG TRAVIS_OS_NAME
ARG TRAVIS_PULL_REQUEST
ARG TRAVIS_PULL_REQUEST_SHA
ARG TRAVIS_REPO_SLUG
ARG TRAVIS_TAG
ARG DRONE
ARG DRONE_TAG
ARG DRONE_COMMIT
ARG DRONE_BRANCH
ARG DRONE_PULL_REQUEST

ARG SKIP_BACKEND_TEST
ARG MONGO_TEST

WORKDIR /go/src/github.com/umputun/remark/backend
ADD backend /go/src/github.com/umputun/remark/backend
ADD .git /go/src/github.com/umputun/remark/.git

# run tests
RUN \
    if [ -f .mongo ] ; then export MONGO_TEST=$(cat .mongo) ; fi && \
    cd app && \
    if [ -z "$SKIP_BACKEND_TEST" ] ; then go test -covermode=count -coverprofile=/profile.cov ./... ; \
    else echo "skip backend test" ; fi

RUN echo "mongo=${MONGO_TEST}" >> /etc/hosts

# linters
RUN if [ -z "$SKIP_BACKEND_TEST" ] ; then \
    if [ -f .mongo ] ; then export MONGO_TEST=$(cat .mongo) ; fi && \
    gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=errcheck --enable=unconvert \
    --enable=deadcode  --enable=gosimple --exclude=test --exclude=mock --exclude=vendor ./... ; \
    else echo "skip backend linters" ; fi


# submit coverage to coverals if COVERALLS_TOKEN in env
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo "coverall not enabled" ; \
    else goveralls -coverprofile=/profile.cov -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi

# if DRONE presented use DRONE_* git env to make version
RUN \
    if [ -z "$DRONE" ] ; then \
    echo "runs outside of drone" && version="local"; \
    else version=${DRONE_TAG}${DRONE_BRANCH}${DRONE_PULL_REQUEST}-${DRONE_COMMIT:0:7}-$(date +%Y%m%d-%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o remark42 -ldflags "-X main.revision=${version} -s -w" ./app


FROM node:10.11-alpine as build-frontend-deps

ARG CI

RUN apk add --no-cache --update git
ADD web/package.json /srv/web/package.json
ADD web/package-lock.json /srv/web/package-lock.json
RUN cd /srv/web && CI=true npm ci

FROM node:10.11-alpine as build-frontend

ARG CI
ARG SKIP_FRONTEND_TEST
ARG NODE_ENV=production

COPY --from=build-frontend-deps /srv/web/node_modules /srv/web/node_modules
ADD web /srv/web
RUN cd /srv/web && \
    if [ -z "$SKIP_FRONTEND_TEST" ] ; then npx run-p lint test build ; \
    else echo "skip frontend tests and lint" ; npm run build ; fi && \
    rm -rf ./node_modules


FROM umputun/baseimage:app-latest

WORKDIR /srv

ADD entrypoint.sh /entrypoint.sh
ADD backend/scripts/backup.sh /usr/local/bin/backup
ADD backend/scripts/restore.sh /usr/local/bin/restore
ADD backend/scripts/import.sh /usr/local/bin/import
RUN chmod +x /entrypoint.sh /usr/local/bin/backup /usr/local/bin/restore /usr/local/bin/import

COPY --from=build-backend /go/src/github.com/umputun/remark/backend/remark42 /srv/remark42
COPY --from=build-frontend /srv/web/public/ /srv/web
RUN chown -R app:app /srv
RUN ln -s /srv/remark42 /usr/bin/remark42

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1

CMD ["server"]
ENTRYPOINT ["/entrypoint.sh"]
