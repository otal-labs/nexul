# Build context: ./web (see docker-compose.yml).
# Static output, so build on the host platform rather than under emulation.
FROM --platform=$BUILDPLATFORM oven/bun:alpine AS build
WORKDIR /app
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile || bun install
COPY . .
# Relative API base: nginx proxies /api, /auth, /ws, /mcp to the server, so
# the SPA talks to the same origin it was served from.
RUN VITE_API_URL=/ bun run build

FROM nginx:alpine AS release
ARG VERSION=dev
LABEL org.opencontainers.image.version=${VERSION}
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
