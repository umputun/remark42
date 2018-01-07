FROM umputun/baseimage:buildgo-latest as build-backend

ADD . /go/src/github.com/umputun/remark
WORKDIR /go/src/github.com/umputun/remark

RUN cd app && go test -v $(go list -e ./... | grep -v vendor)

RUN gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode  --enable=gosimple --enable=gas --exclude=test --exclude=mock --exclude=vendor ./...

#RUN /script/checkvendor.sh
RUN mkdir -p target && /script/coverage.sh

RUN go build -o remark -ldflags "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S) -s -w" ./app


FROM umputun/baseimage:micro-latest

COPY --from=build-backend /go/src/github.com/umputun/remark/remark /srv/
COPY --from=build-backend /go/src/github.com/umputun/remark/web /srv/web
RUN chown -R umputun:umputun /srv

USER umputun

WORKDIR /srv
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD curl --fail http://localhost:8080/ping || exit 1

CMD ["/srv/remark"]
ENTRYPOINT ["/init.sh"]
