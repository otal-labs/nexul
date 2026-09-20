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
  const wholeAppImport = 20_000;

  it("renders the home page through the router", async () => {
    const { App } = await import("./App");
    render(<App />);
    expect(
      await screen.findByRole("heading", { name: "One button. The trail shows every step the agent took." }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in with GitHub" })).toBeInTheDocument();
  }, wholeAppImport);

  it("shows the instance bootstrap page instead of the router when unconfigured", async () => {
    mocks.get.mockResolvedValue({ data: { configured: false } });
    const { App } = await import("./App");
    render(<App />);
    expect(await screen.findByText(/set up this instance/i)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Sign in with GitHub" })).not.toBeInTheDocument();
  }, wholeAppImport);
});
