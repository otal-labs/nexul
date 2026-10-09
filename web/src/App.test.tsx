import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: vi.fn(() => "Something went wrong"),
  joinAPIURL: (path: string) => `http://localhost:8080${path}`,
}));

describe("App", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.get.mockResolvedValue({ data: { configured: true } });
    // App owns a module-scoped QueryClient with a 30s staleTime, so it must be reimported per test to avoid serving the previous test's cached response.
    vi.resetModules();
  });

  // Each test re-imports the whole app graph (vi.resetModules above); under full-suite load that alone can pass vitest's 5s default.
  const wholeAppImport = 45_000;

  it("renders the home page through the router", async () => {
    const { App } = await import("./App");
    render(<App />);
    expect(
      await screen.findByRole("heading", { name: "Docs, tickets, chat, and deploys in one place." }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in" })).toBeInTheDocument();
  }, wholeAppImport);

  it("shows the setup code screen instead of the router when unconfigured", async () => {
    mocks.get.mockResolvedValue({ data: { configured: false, setup_open: true } });
    const { App } = await import("./App");
    render(<App />);
    expect(await screen.findByRole("heading", { name: /enter the setup code/i })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Sign in" })).not.toBeInTheDocument();
  }, wholeAppImport);
});
