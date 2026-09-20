import { afterEach, describe, expect, it, vi } from "vitest";

import { buildCollabURL } from "@/lib/collab/url";

const originalWS = import.meta.env.VITE_WS_URL;

afterEach(() => {
  vi.stubEnv("VITE_WS_URL", originalWS);
});

describe("buildCollabURL", () => {
  it("derives the collab endpoint from the api base when VITE_WS_URL is unset", () => {
    vi.stubEnv("VITE_WS_URL", undefined);
    expect(buildCollabURL("doc-1", "tok", "edit")).toBe(
      "ws://localhost:8080/ws/collab/doc-1?mode=edit&token=tok",
    );
  });

  it("prefers the configured VITE_WS_URL", () => {
    vi.stubEnv("VITE_WS_URL", "ws://example.com/ws/collab");
    expect(buildCollabURL("doc-1", "tok", "view")).toBe("ws://example.com/ws/collab?mode=view&token=tok");
  });

  it("passes the session token for WS auth", () => {
    vi.stubEnv("VITE_WS_URL", undefined);
    expect(buildCollabURL("doc-1", "tok/ens", "edit")).toBe(
      "ws://localhost:8080/ws/collab/doc-1?mode=edit&token=tok%2Fens",
    );
  });
});
