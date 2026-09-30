import { afterEach, describe, expect, it, vi } from "vitest";

import { buildLiveURL, buildLogsURL } from "@/lib/live";

const originalWS = import.meta.env.VITE_WS_URL;

afterEach(() => {
  vi.stubEnv("VITE_WS_URL", originalWS);
});

describe("buildLiveURL", () => {
  it("derives the ws/events URL from the api base when VITE_WS_URL is unset", () => {
    vi.stubEnv("VITE_WS_URL", undefined);
    expect(buildLiveURL("tok")).toBe("ws://localhost:8080/ws/events?token=tok");
  });

  it("prefers the configured VITE_WS_URL", () => {
    vi.stubEnv("VITE_WS_URL", "ws://example.com/ws/events");
    expect(buildLiveURL("tok")).toBe("ws://example.com/ws/events?token=tok");
  });

  it("appends the token as a query param", () => {
    vi.stubEnv("VITE_WS_URL", undefined);
    expect(buildLiveURL("tok/ens")).toBe("ws://localhost:8080/ws/events?token=tok%2Fens");
  });
});

describe("buildLogsURL", () => {
  it("addresses the service's logs socket with the tail and the token", () => {
    expect(buildLogsURL("tok/ens", "s-1", "web app", 20)).toBe(
      "ws://localhost:8080/ws/stacks/s-1/services/web%20app/logs?tail=20&token=tok%2Fens",
    );
  });
});
