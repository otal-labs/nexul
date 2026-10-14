import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { RunnersPage } from "@/pages/RunnersPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), holds: [] as string[] }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));
vi.mock("@/hooks/AccessHooks", () => ({ useHasInstancePermission: (value: string) => mocks.holds.includes(value) }));

const renderPage = () => {
  mocks.get.mockImplementation((url: string) =>
    Promise.resolve({ data: url === "/api/runners/latest-version" ? { version: "v0.1.6" } : [] }),
  );
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient()}>
        <RunnersPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
};

describe("RunnersPage", () => {
  it("renders the page heading and panels, with Add runner for someone who may add machines", async () => {
    mocks.holds = ["machines:create"];
    renderPage();
    expect(screen.getByRole("heading", { level: 1, name: "Runners" })).toBeInTheDocument();
    expect(await screen.findByText("Waiting for a runner")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Add runner/ })).toBeInTheDocument();
  });

  it("offers no Add runner to someone who may not add machines", async () => {
    mocks.holds = [];
    renderPage();
    expect(await screen.findByText("Waiting for a runner")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Add runner/ })).not.toBeInTheDocument();
  });
});
