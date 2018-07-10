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
ARG MONGO

WORKDIR /go/src/github.com/umputun/remark/backend
ADD backend /go/src/github.com/umputun/remark/backend

# run tests
RUN cd app && \
    echo "$MONGO mongo" >> /etc/hosts && \
    if [ -z "$SKIP_BACKEND_TEST" ] ; then go test ./... ; \
    else echo "skip backend test" ; fi

# linters
RUN if [ -z "$SKIP_BACKEND_TEST" ] ; then \
    gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode  --enable=gosimple --enable=gas --exclude=test --exclude=mock --exclude=vendor ./... ; \
    else echo "skip backend linters" ; fi

# coverage report
RUN if [ -z "$SKIP_BACKEND_TEST" ] ; then \
    echo "$MONGO mongo" >> /etc/hosts && \
    mkdir -p target && /script/coverage.sh ; \
    else echo "skip backend coverage" ; fi

# submit coverage to coverals if COVERALLS_TOKEN in env
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo "coverall not enabled" ; \
    else goveralls -coverprofile=.cover/cover.out -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi

# if DRONE presented use DRONE_* git env to make version
RUN \
    if [ -z "$DRONE" ] ; then \
    echo "runs outside of drone" && version="local"; \
    else version=${DRONE_TAG}${DRONE_BRANCH}${DRONE_PULL_REQUEST}-${DRONE_COMMIT:0:7}-$(date +%Y%m%d-%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o remark -ldflags "-X main.revision=${version} -s -w" ./app


FROM node:9.4-alpine as build-frontend

ARG CI
ARG SKIP_FRONTEND_TEST

ADD web /srv/web
RUN apk add --no-cache --update git
RUN \
    cd /srv/web && npm i && \
    if [ -z "$SKIP_FRONTEND_TEST" ] ; then npm run lint && npm run test ; \
    else echo "skip frontend tests and lint" ; fi && \ 
    npm run build && rm -rf ./node_modules


FROM umputun/baseimage:app-latest

WORKDIR /srv

ADD backend/scripts/*.sh /srv/
ADD start.sh /srv/start.sh
RUN chmod +x /srv/*.sh

COPY --from=build-backend /go/src/github.com/umputun/remark/backend/remark /srv/
COPY --from=build-frontend /srv/web/public/ /srv/web
RUN chown -R app:app /srv

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1

CMD ["/srv/start.sh"]
ENTRYPOINT ["/init.sh"]
