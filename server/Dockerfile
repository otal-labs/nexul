# Builds the server image from source, web UI embedded (used by docker-compose.e2e.yml). Build context: ./server;
# the `root` named context supplies the Go module, internal/, and web/. Releases use Dockerfile.release instead.
FROM --platform=$BUILDPLATFORM oven/bun:alpine AS web
WORKDIR /web
COPY --from=root web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY --from=root web/ ./
RUN VITE_API_URL=/ bun run build

# Cross-compile from the build host: an emulated arm64 Go build under QEMU takes many times longer.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY --from=root go.mod go.sum ./
RUN go mod download
COPY --from=root *.go docker-compose.yml ./
COPY --from=root internal/ ./internal/
COPY . ./server/
COPY --from=web /web/dist ./server/webui/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -tags embed -trimpath -ldflags="-s -w -X github.com/otal-labs/nexul/internal/platform/version.Version=${VERSION}" -o /nexul ./server/cmd/

FROM alpine:3.22 AS release
ARG VERSION=dev
LABEL org.opencontainers.image.version=${VERSION}
COPY --from=build /nexul /usr/local/bin/nexul
EXPOSE 8080
ENTRYPOINT ["nexul"]
CMD ["serve"]
