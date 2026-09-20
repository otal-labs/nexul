import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationKind, AutomationVersionStatus } from "@/enums/Automation";
import { AutomationPage } from "@/pages/AutomationPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const automation = {
  id: "a1",
  name: "Ticket finished",
  description: "Moves a ticket to done",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: ["ticket.pr_merged"],
  config_schema: { fields: [] },
  config_values: {},
  scopes: ["tickets:write"],
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
};

const routeFor = (endpoints: Record<string, unknown>) => (url: string) => {
  for (const [path, data] of Object.entries(endpoints)) {
    if (url === path) return Promise.resolve({ data });
  }
  return Promise.reject(new Error(`unhandled GET ${url}`));
};

const renderPage = (endpoints: Record<string, unknown>) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  mocks.get.mockImplementation(routeFor(endpoints));
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/automations/a1"]}>
        <Routes>
          <Route path="/automations/:id" element={<AutomationPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("AutomationPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  const baseEndpoints = {
    "/api/automations/a1": automation,
    "/api/automations/a1/runs": [],
    "/api/workspaces/ws-1/me": { role_name: "Owner", permissions: ["automations:write", "automations:delete"] },
    "/api/automations/a1/versions/diff": { active: null, pending: null },
  };

  it("renders the automation's name, subscriptions, and empty run history", async () => {
    renderPage(baseEndpoints);

    expect(await screen.findByText("Ticket finished")).toBeInTheDocument();
    expect(screen.getByText("ticket.pr_merged")).toBeInTheDocument();
    expect(screen.getByText("No runs yet")).toBeInTheDocument();
  });

  it("shows the pending-version banner when a version is pending", async () => {
    renderPage({
      ...baseEndpoints,
      "/api/automations/a1/versions/diff": {
        active: { id: "v1", automation_id: "a1", sequence: 1, code: "a", pusher_id: "u1", status: AutomationVersionStatus.Active, created_at: "2026-08-01T00:00:00Z" },
        pending: { id: "v2", automation_id: "a1", sequence: 2, code: "b", pusher_id: "u1", status: AutomationVersionStatus.Pending, created_at: "2026-08-02T00:00:00Z" },
      },
    });

    expect(await screen.findByText(/pending review/)).toBeInTheDocument();
  });

  it("switches to the Versions tab and loads version history", async () => {
    const user = userEvent.setup();
    renderPage({
      ...baseEndpoints,
      "/api/automations/a1/versions": [
        { id: "v1", automation_id: "a1", sequence: 1, code: "a", pusher_id: "u1", status: AutomationVersionStatus.Active, created_at: "2026-08-01T00:00:00Z" },
      ],
    });

    await screen.findByText("Ticket finished");
    await user.click(screen.getByRole("tab", { name: "Versions" }));

    expect(await screen.findByText("#1")).toBeInTheDocument();
  });
});
