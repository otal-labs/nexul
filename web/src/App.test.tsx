import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { App } from "@/App";
import { queryClient } from "@/lib/queryClient";

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
    // App owns a module-scoped QueryClient with a 30s staleTime; clearing it keeps one test's cached bootstrap answer from serving the next.
    queryClient.clear();
  });

  it("renders the home page through the router", async () => {
    render(<App />);
    expect(
      await screen.findByRole("heading", { name: "Docs, tickets, chat, and deploys in one place." }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in" })).toBeInTheDocument();
  });

  it("shows the setup code screen instead of the router when unconfigured", async () => {
    mocks.get.mockResolvedValue({ data: { configured: false, setup_open: true } });
    render(<App />);
    expect(await screen.findByRole("heading", { name: /enter the setup code/i })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Sign in" })).not.toBeInTheDocument();
  });
});
