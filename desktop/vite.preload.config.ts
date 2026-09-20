// Sandboxed preloads can only require electron, not relative files/npm packages, so this bundles to one CJS file.
import { defineConfig } from "vite";

export default defineConfig({
  build: {
    lib: {
      entry: "electron/preload.ts",
      formats: ["cjs"],
      fileName: () => "preload.js",
    },
    outDir: "dist-electron",
    emptyOutDir: false,
    rollupOptions: {
      external: ["electron"],
    },
  },
});
