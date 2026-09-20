# Build context: ./automations (see docker-compose.yml). The `root`
# named context supplies sdk/ so package.json's file:../sdk dependency
# resolves inside the image the same way it does on a checkout.
FROM oven/bun:1-slim AS release
ARG VERSION=dev
LABEL org.opencontainers.image.version=${VERSION}
WORKDIR /app
COPY --from=root sdk/ /sdk/
COPY package.json bun.lock* ./
RUN bun install --production
COPY src/ ./src/
ENTRYPOINT ["bun", "run", "src/main.ts"]
