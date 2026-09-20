import { afterEach, describe, expect, it, vi } from "vitest";

// The URL builders are evaluated at module scope, so each test re-imports the module with a fresh environment (vi.resetModules + dynamic import).

const load = async () => {
  vi.resetModules();
  return import("@/api/client");
};

const unstub = () => {
  vi.unstubAllEnvs();
  vi.resetModules();
};

describe("joinAPIURL", () => {
  afterEach(unstub);

  it("does not double-slash a same-origin base (VITE_API_URL=/)", async () => {
    vi.stubEnv("VITE_API_URL", "/");
    const { joinAPIURL } = await load();
    expect(joinAPIURL("/auth/github")).toBe("/auth/github");
  });

  it("keeps an absolute base intact", async () => {
    vi.stubEnv("VITE_API_URL", "http://localhost:8080");
    const { joinAPIURL } = await load();
    expect(joinAPIURL("/auth/github")).toBe("http://localhost:8080/auth/github");
  });

  it("trims a trailing slash on an absolute base", async () => {
    vi.stubEnv("VITE_API_URL", "https://api.example.com/");
    const { joinAPIURL } = await load();
    expect(joinAPIURL("/auth/github")).toBe("https://api.example.com/auth/github");
  });
});

describe("resolveWSBase", () => {
  afterEach(unstub);

  it("derives the page host for same-origin builds (no //ws/… host bug)", async () => {
    vi.stubEnv("VITE_API_URL", "/");
    const { resolveWSBase } = await load();
    expect(resolveWSBase()).toBe(`ws://${window.location.host}`);
  });

  it("swaps the scheme for an absolute API base", async () => {
    vi.stubEnv("VITE_API_URL", "https://api.example.com");
    const { resolveWSBase } = await load();
    expect(resolveWSBase()).toBe("wss://api.example.com");
  });
});
