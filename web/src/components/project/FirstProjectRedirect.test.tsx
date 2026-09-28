import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { FirstProjectRedirect } from "@/components/project/FirstProjectRedirect";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(),
}));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const mockApi = (canCreate: boolean, projects: unknown[]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: { can_create_workspace: canCreate } } };
    if (url === "/api/projects") return { data: projects };
    throw new Error(`unexpected GET ${url}`);
  });

const renderAtHome = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route
            path="/"
            element={
              <>
                <FirstProjectRedirect />
                <p>Home</p>
              </>
            }
          />
          <Route path="/wizard/project/project" element={<p>Project wizard</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("FirstProjectRedirect", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "workspace-default" });
  });

  it("stays put for someone who cannot create a project", async () => {
    mockApi(false, []);
    renderAtHome();

    await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/projects", expect.anything()));
    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.queryByText("Project wizard")).not.toBeInTheDocument();
  });

  it("stays put once the workspace has a project", async () => {
    mockApi(true, [project]);
    renderAtHome();

    await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/projects", expect.anything()));
    expect(screen.getByText("Home")).toBeInTheDocument();
  });

  it("waits for a workspace before deciding", () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "" });
    mockApi(true, []);
    renderAtHome();

    expect(api.get).not.toHaveBeenCalledWith("/api/projects", expect.anything());
    expect(screen.getByText("Home")).toBeInTheDocument();
  });

  it("sends an owner with no project to the project wizard", async () => {
    mockApi(true, []);
    renderAtHome();

    expect(await screen.findByText("Project wizard")).toBeInTheDocument();
  });
});
