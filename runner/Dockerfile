# The containerized runner the e2e stack uses; releases ship the runner as a host binary instead.
# Build context: ./runner; the `root` named context supplies the shared Go module and internal/ packages.
# Cross-compile from the build host: an emulated arm64 Go build under QEMU takes many times longer.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY --from=root go.mod go.sum ./
RUN go mod download
COPY --from=root internal/ ./internal/
COPY . ./runner/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X github.com/otal-labs/nexul/internal/platform/version.Version=${VERSION}" -o /runner ./runner/cmd/

FROM alpine:3.20 AS release
ARG VERSION=dev
LABEL org.opencontainers.image.version=${VERSION}
# git: repo-driven builds clone on the runner; docker-cli/compose: every deploy shells out to docker.
RUN apk add --no-cache ca-certificates git docker-cli docker-compose
COPY --from=build /runner /usr/local/bin/runner
ENTRYPOINT ["runner"]
