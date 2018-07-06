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
ARG DRONE_TAG
ARG DRONE_COMMIT

WORKDIR /go/src/github.com/umputun/remark/backend

ADD backend /go/src/github.com/umputun/remark/backend
ADD .git /go/src/github.com/umputun/remark/.git

RUN cd app && go test ./...

RUN gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode  --enable=gosimple --enable=gas --exclude=test --exclude=mock --exclude=vendor ./...

# coverage test, submit to coverals if COVERALLS_TOKEN in env
RUN mkdir -p target && /script/coverage.sh
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo coverall not enabled ; \
    else goveralls -coverprofile=.cover/cover.out -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi

# get revision from git. if DRONE_TAG presented use DRONE_* git env to make version
RUN \
    cd /go/src/github.com/umputun/remark && version=$(/script/git-rev.sh) && cd backend \
    echo "git version=$version" && \  
    if [ -z "$DRONE_TAG" ] ; then \
    echo "runs outside of drone" ; \
    else version=${DRONE_TAG}-${DRONE_COMMIT:0:7}-$(date +%Y%m%d-%H:%M:%S); fi && \
    echo "final version=$version" && \  
    go build -o remark -ldflags "-X main.revision=${version} -s -w" ./app


FROM node:9.4-alpine as build-frontend

ADD web /srv/web
RUN apk add --no-cache --update git
RUN \
    cd /srv/web && \
    npm i && npm run lint && npm run test && npm run build && \
    rm -rf ./node_modules


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
