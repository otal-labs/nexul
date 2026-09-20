# E2E test image: Bun + Playwright chromium + the web source and specs.
# Build context: ./web (see docker-compose.e2e.yml).
FROM oven/bun:1

WORKDIR /e2e
COPY package.json bun.lock* ./
RUN bun install
COPY . .
# Browser matching the installed playwright version (handles system deps).
RUN bunx playwright install --with-deps chromium

CMD ["bun", "run", "test:e2e"]
