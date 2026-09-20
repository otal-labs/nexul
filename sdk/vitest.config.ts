import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    include: ["test/**/*.test.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text"],
      exclude: ["bin/**", "tools/**", "src/events.generated.ts", "**/*.config.*"],
    },
  },
});
