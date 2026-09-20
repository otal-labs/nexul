import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoriesPage } from "@/pages/MemoriesPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: (error: unknown) => (error as Error)?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const memory = {
  id: "mem-1",
  workspace_id: "ws-1",
  project_id: "p-1",
  title: "Deploy quirks",
  when_to_use: "use this if touching deploy config",
  body: "body",
  always_included: true,
  created_by: "user-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "user-1",
  updated_at: "2026-09-16T12:00:00Z",
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
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter>
        <MemoriesPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const baseEndpoints = {
  "/api/projects": [project],
  "/api/workspaces/ws-1/me": { role_name: "Owner", permissions: ["memories:write"] },
};

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

const workspaceMemory = {
  ...memory,
  id: "mem-2",
  project_id: "",
  title: "House style",
};

describe("MemoriesPage", () => {
  it("renders memories grouped under their project", async () => {
    renderPage({ ...baseEndpoints, "/api/memories": [memory] });
    expect(await screen.findByText("Deploy quirks")).toBeInTheDocument();
    expect(screen.getByText("Backend")).toBeInTheDocument();
    expect(screen.getByText("always included")).toBeInTheDocument();
  });

  it("groups workspace memories under a Workspace heading before per-project groups", async () => {
    renderPage({ ...baseEndpoints, "/api/memories": [memory, workspaceMemory] });
    await screen.findByText("Deploy quirks");
    const headings = screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);
    expect(headings).toEqual(["Workspace", "Backend"]);
  });

  it("shows the New memory action when the actor can write", async () => {
    renderPage({ ...baseEndpoints, "/api/memories": [memory] });
    expect(await screen.findByRole("button", { name: "New memory" })).toBeInTheDocument();
  });

  it("hides the New memory action without memories:write", async () => {
    renderPage({
      ...baseEndpoints,
      "/api/memories": [memory],
      "/api/workspaces/ws-1/me": { role_name: "Member", permissions: [] },
    });
    await screen.findByText("Deploy quirks");
    expect(screen.queryByRole("button", { name: "New memory" })).not.toBeInTheDocument();
  });

  it("shows the shared empty state when there are no memories", async () => {
    renderPage({ ...baseEndpoints, "/api/memories": [] });
    expect(await screen.findByText("No memories yet.")).toBeInTheDocument();
  });

  it("shows the shared error display when the list fails", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/memories") return Promise.reject(new Error("boom"));
      return routeFor(baseEndpoints)(url);
    });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <MemoriesPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByText("boom")).toBeInTheDocument();
  });
});
