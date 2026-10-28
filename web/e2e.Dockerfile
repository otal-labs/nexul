# E2E test image: Bun + Playwright chromium + the web source and specs.
# Build context: ./web (see docker-compose.e2e.yml).
FROM oven/bun:1
# Vitest runs on Node; without it bunx falls back to the Bun runtime, which Vitest does not support.
COPY --from=node:24-bookworm-slim /usr/local/bin/node /usr/local/bin/node

WORKDIR /e2e
COPY package.json bun.lock* ./
RUN bun install
COPY . .
# Browser matching the installed playwright version (handles system deps).
RUN bunx playwright install --with-deps chromium

CMD ["bun", "run", "test:e2e"]
