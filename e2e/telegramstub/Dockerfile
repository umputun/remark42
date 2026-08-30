# The stub is stdlib only, so it is compiled straight from the one source file with a throwaway
# module. Building it as part of the e2e module instead would pull playwright-go and its browser
# tooling into an image that needs none of it.
FROM golang:1.25-alpine AS build

WORKDIR /src
COPY main.go .
RUN go mod init telegramstub >/dev/null && \
    CGO_ENABLED=0 go build -trimpath -o /telegramstub .

FROM alpine:3.22

RUN apk add --no-cache curl
COPY --from=build /telegramstub /telegramstub

EXPOSE 8091
CMD ["/telegramstub"]
