import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

const DEV_PROXY_HOST = process.env.VITE_PROXY_HOST || "localhost";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  // The gateway serves the SPA same-origin in production (embedded webui), so
  // the Go server sends no CORS headers. Dev mode therefore proxies the API,
  // OAuth, and the WS through Vite instead of calling localhost:8080 cross-
  // origin. Run with VITE_API_URL=/ (df-14).
  //
  // Forwards the paths the server owns, so the SPA reaches it same-origin as it
  // does in production, where the server serves the SPA itself. Host
  // is overridable because bare `bun run dev` needs localhost while the
  // dockerized debug stack needs the `server` service name.
  server: {
    host: true,
    // The debug stack is reached through a Cloudflare tunnel or proxy under a real hostname; Vite 403s unknown hosts by default.
    allowedHosts: true,
    proxy: {
      "/api": `http://${DEV_PROXY_HOST}:8080`,
      "/auth": `http://${DEV_PROXY_HOST}:8080`,
      "/hooks": `http://${DEV_PROXY_HOST}:8080`,
      "/mcp": `http://${DEV_PROXY_HOST}:8080`,
      "/ws": { target: `ws://${DEV_PROXY_HOST}:8080`, ws: true },
    },
  },
});
