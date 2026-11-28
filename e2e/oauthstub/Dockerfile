# Stdlib only, compiled straight from the one source file with a throwaway module, as the
# telegram stub is.
FROM golang:1.27-alpine AS build

WORKDIR /src
COPY main.go .
RUN go mod init oauthstub >/dev/null && \
    CGO_ENABLED=0 go build -trimpath -o /oauthstub .

FROM alpine:3.22

RUN apk add --no-cache curl
COPY --from=build /oauthstub /oauthstub

EXPOSE 8092
CMD ["/oauthstub"]
