import { defineConfig } from "vitest/config";

// End-to-end config: runs the browser (Playwright) and API/MCP specs against
// a live stack, inside the e2e docker network (see docker-compose.e2e.yml).
// Separate from the unit config so `bun run test` stays fast and hermetic.
export default defineConfig({
  test: {
    // Specs live under e2e/; unit tests are *.test.tsx, so *.spec.ts only
    // matches the e2e suite. Use a rootless pattern — a config-dir-relative
    // prefix resolves differently inside the container. Note: do NOT set
    // fileParallelism here — vitest 3.2.x fails test discovery with it.
    include: ["**/*.spec.ts"],
    globalSetup: ["./e2e/setup/global.ts"],
    environment: "node",
    // One worker so spec files don't interleave against the shared stack
    // (live events from one spec's deploys re-render another spec's pages).
    // Files run in parallel (vitest 3.2.x breaks test discovery with any
    // serialization setting — fileParallelism:false, maxForks:1 and
    // threads.singleThread all fail). Specs must not mutate shared user
    // state; the onboarding spec (which flips wizard flags) is excluded
    // here and run as its own phase after the main suite.
    pool: "forks",
    testTimeout: 60_000,
    hookTimeout: 60_000,
  },
});
