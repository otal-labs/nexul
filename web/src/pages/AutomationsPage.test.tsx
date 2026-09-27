import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationsPage } from "@/pages/AutomationsPage";
import { AutomationKind } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, patch: mocks.patch },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const automation = (overrides: Partial<Automation> = {}): Automation => ({
  id: "a1",
  name: "Ticket finished",
  description: "Moves a ticket to done once every linked PR is merged",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: [],
  config_schema: {},
  config_values: {},
  scopes: ["tickets:write"],
  host_id: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/automations"]}>
        <AutomationsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("AutomationsPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.patch.mockReset();
  });

  it("shows an empty state with no automations", async () => {
    mocks.get.mockResolvedValue({ data: [] });
    renderPage();

    expect(await screen.findByText("No automations yet")).toBeInTheDocument();
    expect(await screen.findByText("No automations host enrolled yet.")).toBeInTheDocument();
  });

  it("lists automations and the automations hosts returned by the API", async () => {
    const host = {
      id: "h1", name: "jobs-1", machine: "prod", os: "linux", arch: "amd64", version: "v0.3.0",
      connected: true, last_seen: "2026-08-01T00:00:00Z",
    };
    mocks.get.mockImplementation(async (url: string) =>
      url === "/api/automation-hosts"
        ? { data: [host] }
        : { data: [automation(), automation({ id: "a2", name: "PR opened" })] },
    );
    renderPage();

    expect(await screen.findByText("Ticket finished")).toBeInTheDocument();
    expect(screen.getByText("PR opened")).toBeInTheDocument();
    expect(await screen.findByText("jobs-1")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add automations host" })).toBeInTheDocument();
  });
});
