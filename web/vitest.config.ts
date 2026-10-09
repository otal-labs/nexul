import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@nexul/client-core": path.resolve(__dirname, "../client-core"),
    },
  },
  // The modules shared with the phone app live beside web/, and their tests run here, once.
  server: { fs: { allow: [".", "../client-core"] } },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    // Only the stylesheet guard reads index.css, as raw text; every other import still gets an empty module.
    css: { include: [/src\/index\.css/] },
    include: ["src/**/*.test.{ts,tsx}", "../client-core/**/*.test.ts"],
    // e2e specs run under vitest.e2e.config.ts inside the e2e docker network.
    exclude: ["e2e/**", "node_modules/**"],
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov"],
      reportsDirectory: "./coverage",
      thresholds: {
        lines: 80,
        branches: 80,
        functions: 80,
        statements: 80,
      },
      exclude: [
        "e2e/**",
        "src/components/ui/**",
        "src/lib/utils.ts",
        "src/test/**",
        "dist/**",
        "coverage/**",
        "**/*.config.*",
        "**/*.d.ts",
        "src/main.tsx",
      ],
    },
  },
});
