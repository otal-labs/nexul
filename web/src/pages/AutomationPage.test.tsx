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
  host_id: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
};

const routeFor = (endpoints: Record<string, unknown>) => (url: string) => {
  for (const [path, data] of Object.entries(endpoints)) {
    if (url === path) return Promise.resolve({ data });
  }
  return Promise.reject(new Error(`unhandled GET ${url}`));
};

const renderPage = (endpoints: Record<string, unknown>, route = "/automations/a1") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  mocks.get.mockImplementation(routeFor(endpoints));
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/automations/:id/:tab?" element={<AutomationPage />} />
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
    "/api/automation-hosts": [],
  };

  const pendingDiff = {
    active: { id: "v1", automation_id: "a1", sequence: 1, code: "a", pusher_id: "u1", status: AutomationVersionStatus.Active, created_at: "2026-08-01T00:00:00Z" },
    pending: { id: "v2", automation_id: "a1", sequence: 2, code: "b", pusher_id: "u1", status: AutomationVersionStatus.Pending, created_at: "2026-08-02T00:00:00Z" },
  };
  const versions = [
    { id: "v1", automation_id: "a1", sequence: 1, code: "a", pusher_id: "u1", status: AutomationVersionStatus.Active, created_at: "2026-08-01T00:00:00Z" },
  ];

  it("shows the run history error on the Runs tab", async () => {
    renderPage(Object.fromEntries(Object.entries(baseEndpoints).filter(([path]) => path !== "/api/automations/a1/runs")));

    expect(await screen.findByText("Ticket finished")).toBeInTheDocument();
    expect(await screen.findByText("error")).toBeInTheDocument();
  });

  it("hides the Danger zone tab without automations:delete, even when the path asks for it", async () => {
    renderPage(
      { ...baseEndpoints, "/api/workspaces/ws-1/me": { role_name: "Viewer", permissions: [] } },
      "/automations/a1/danger",
    );

    expect(await screen.findByText("No runs yet")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Danger zone" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete automation/i })).not.toBeInTheDocument();
  });

  it("opens on Runs, then shows subscriptions on Configuration and the delete action on Danger zone", async () => {
    const user = userEvent.setup();
    renderPage(baseEndpoints);

    expect(await screen.findByText("Ticket finished")).toBeInTheDocument();
    expect(await screen.findByText("No runs yet")).toBeInTheDocument();
    expect(screen.queryByText("ticket.pr_merged")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Configuration" }));
    expect(screen.getByText("ticket.pr_merged")).toBeInTheDocument();

    await user.click(await screen.findByRole("tab", { name: "Danger zone" }));
    expect(screen.getByRole("button", { name: /delete automation/i })).toBeInTheDocument();
  });

  it("links the pending-version banner on Configuration to the Versions tab", async () => {
    const user = userEvent.setup();
    renderPage(
      { ...baseEndpoints, "/api/automations/a1/versions/diff": pendingDiff, "/api/automations/a1/versions": versions },
      "/automations/a1/configuration",
    );

    await user.click(await screen.findByText(/waiting to be merged/));

    expect(await screen.findByRole("tab", { name: "Versions", selected: true })).toBeInTheDocument();
    expect(await screen.findByText("#1")).toBeInTheDocument();
  });

  it("opens /versions directly on version history", async () => {
    renderPage({ ...baseEndpoints, "/api/automations/a1/versions": versions }, "/automations/a1/versions");

    expect(await screen.findByText("#1")).toBeInTheDocument();
  });

  it("switches to the Versions tab and loads version history", async () => {
    const user = userEvent.setup();
    renderPage({ ...baseEndpoints, "/api/automations/a1/versions": versions });

    await screen.findByText("Ticket finished");
    await user.click(screen.getByRole("tab", { name: "Versions" }));

    expect(await screen.findByText("#1")).toBeInTheDocument();
  });
});
