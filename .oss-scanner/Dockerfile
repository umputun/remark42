# image for anthropic's oss scanner. build from the repo root:
#   docker build -f .oss-scanner/Dockerfile .
# the root .dockerignore is a whitelist, so /src holds only backend/ and frontend/apps/remark42/;
# the scanner adds .git after the build. network is available during the build only; everything
# a test needs must be fetched here.
FROM golang:1.27.2-bookworm AS go-toolchain

FROM node:24-bookworm
COPY --from=go-toolchain /usr/local/go /usr/local/go

# CI=true skips the husky hook install in the widget's prepare script, which needs a .git
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}" \
    DEBIAN_FRONTEND=noninteractive \
    CI=true \
    CGO_ENABLED=1 \
    GOTOOLCHAIN=local

RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential ca-certificates curl git jq procps \
    && rm -rf /var/lib/apt/lists/*

RUN npm i -g pnpm@10.10.0

COPY . /src
WORKDIR /src

RUN cd frontend/apps/remark42 && pnpm install --frozen-lockfile && pnpm build

# the backend embeds the built widget from app/cmd/web
RUN cp -r frontend/apps/remark42/public/. backend/app/cmd/web/ \
    && cd backend && go build -o /usr/local/bin/remark42 ./app

# backend tests compare timestamps built with time.Local and expect a non-UTC zone, as CI sets
ENV TZ=America/Chicago

RUN cd backend/app && (go test -short -count=1 -timeout=300s ./... \
    || echo 'WARNING: backend baseline tests failed')
RUN cd frontend/apps/remark42 && (pnpm test || echo 'WARNING: frontend baseline tests failed')

# backend/vendor is committed, so go builds and tests run offline from it
ENV GOFLAGS=-mod=vendor \
    GOPROXY=off \
    GOSUMDB=off

CMD ["bash"]
