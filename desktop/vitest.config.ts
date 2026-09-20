import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    setupFiles: ["./test/setup.ts"],
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
        "electron/main.ts",
        "electron/preload.ts",
        "electron/ipc.ts",
        "src/main.tsx",
        "**/*.d.ts",
        "dist/**",
        "dist-electron/**",
        "release/**",
      ],
    },
  },
});
