# Build context: ./server (see docker-compose.yml). The `root` named context
# supplies the shared Go module (go.mod/go.sum) and internal/ packages; this
# context supplies server/cmd and server/webui.
# Cross-compile from the build host: an emulated arm64 Go build under QEMU takes many times longer.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY --from=root go.mod go.sum ./
RUN go mod download
COPY --from=root internal/ ./internal/
COPY . ./server/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X github.com/otal-labs/nexul/internal/platform/version.Version=${VERSION}" -o /server ./server/cmd/

FROM alpine:3.20 AS release
ARG VERSION=dev
LABEL org.opencontainers.image.version=${VERSION}
# CA bundle copied from the Go stage: no RUN here, so the arm64 image needs no emulation at all.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /server /usr/local/bin/server
EXPOSE 8080 8081
ENTRYPOINT ["server"]
