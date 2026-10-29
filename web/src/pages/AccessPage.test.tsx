import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { Link, RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AccessPage } from "@/pages/AccessPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { AccessGrid } from "@/models/AccessGrid";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), toast: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get, put: mocks.put }, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: mocks.toast, error: vi.fn() } }));
vi.mock("@/components/settings/RoleSettingsSection", () => ({ RoleSettingsSection: () => <p>Roles card</p> }));

const grid = (over: Partial<AccessGrid> = {}): AccessGrid => ({
  workspace_id: "ws-1",
  setup: { finished: false },
  roles: [
    { id: "r-dev", name: "Developer", holders: 6 },
    { id: "r-client", name: "Client", holders: 5 },
  ],
  projects: [
    { id: "p-sto", name: "Storefront", prefix: "STO", role_ids: ["r-dev"], user_ids: [] },
    { id: "p-mkt", name: "Marketing site", prefix: "MKT", role_ids: [], user_ids: [] },
  ],
  channels: [{ id: "c-backend", name: "backend", kind: "channel", hidden_role_ids: [] }],
  ...over,
});

const me = (owner: boolean, finished: boolean, permissions: string[]) => ({
  role_name: owner ? "Owner" : "Developer",
  roles: [{ id: owner ? "r-owner" : "r-dev", name: owner ? "Owner" : "Developer", is_owner: owner }],
  permissions,
  projects: [],
  access_setup_finished: finished,
});

const serve = (data: AccessGrid, owner = true, permissions = ["projects:access", "roles:write"]) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/access") return { data };
    if (url === "/api/workspaces/ws-1/me") return { data: me(owner, data.setup.finished, permissions) };
    if (url === "/api/workspaces/ws-1/roles") return { data: data.roles };
    if (url === "/api/workspaces") return { data: [{ id: "ws-1", name: "Acme", slug: "acme" }] };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-lena", login: "lena", display_name: "Lena", avatar_url: "" }] } };
    return { data: [] };
  });

const renderPage = (path = "/acme/access") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <RouterProvider
        router={createMemoryRouter(
          [
            { path: "/acme/access/:tab?", element: <><Link to="/acme/chat">Leave</Link><AccessPage /></> },
            { path: "/acme/chat", element: <p>chat page</p> },
          ],
          { initialEntries: [path] },
        )}
      />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  mocks.get.mockReset();
  mocks.put.mockReset();
  mocks.toast.mockReset();
  mocks.put.mockResolvedValue({});
});

describe("AccessPage", () => {
  it("lets the Owner finish the setup with both grids in one save", async () => {
    serve(grid());
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByRole("heading", { name: "Who's in each project" })).toBeInTheDocument();
    expect(screen.getByText("2 projects · 2 roles")).toBeInTheDocument();
    expect(screen.getByText("1 project lists nobody yet")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Client in Marketing site" }));
    await user.click(screen.getByRole("checkbox", { name: "Client in backend" }));
    expect(screen.getByText("Every project lists someone")).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Client in Marketing site" })).toHaveAttribute("data-changed");
    expect(screen.getByRole("checkbox", { name: "Client in backend" })).toHaveAttribute("data-changed");
    expect(screen.getByRole("checkbox", { name: "Developer in backend" })).not.toHaveAttribute("data-changed");

    await user.click(screen.getByRole("button", { name: "Finish setup" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/workspaces/ws-1/access", {
      projects: [{ project_id: "p-mkt", role_ids: ["r-client"], user_ids: [] }],
      channels: [{ channel_id: "c-backend", hidden_role_ids: ["r-client"] }],
      finish: true,
    });
    expect(mocks.toast).toHaveBeenCalledWith("Access is set", expect.objectContaining({ description: expect.stringContaining("Access page") }));
  });

  it("finishes with every project left empty", async () => {
    serve(grid({ projects: grid().projects.map((p) => ({ ...p, role_ids: [] })) }));
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Finish setup" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/workspaces/ws-1/access", { projects: [], channels: [], finish: true });
  });

  it("ticks a role on every project from the lead row, and clears it from every one when all have it", async () => {
    serve(grid());
    const user = userEvent.setup();
    renderPage();

    const lead = await screen.findByRole("checkbox", { name: "Developer: every project" });
    expect(lead).toHaveAttribute("data-state", "indeterminate");
    await user.click(lead);
    expect(screen.getByRole("checkbox", { name: "Developer in Marketing site" })).toBeChecked();
    await user.click(lead);
    expect(screen.getByRole("checkbox", { name: "Developer in Storefront" })).not.toBeChecked();
  });

  it("names a person on a project from the People column", async () => {
    serve(grid());
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Add a person to Storefront" }));
    expect(await screen.findByText("People in Acme")).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: /Lena/ }));
    await user.click(screen.getByRole("button", { name: "Finish setup" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/workspaces/ws-1/access", expect.objectContaining({
      projects: [{ project_id: "p-sto", role_ids: ["r-dev"], user_ids: ["u-lena"] }],
    }));
  });

  it("asks before leaving with unsaved ticks, and stays when told to", async () => {
    serve(grid());
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("checkbox", { name: "Client in Storefront" }));
    await user.click(screen.getByRole("link", { name: "Leave" }));
    const prompt = await screen.findByRole("dialog", { name: "Discard changes?" });
    await user.click(within(prompt).getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("checkbox", { name: "Client in Storefront" })).toBeChecked();

    await user.click(screen.getByRole("link", { name: "Leave" }));
    await user.click(within(await screen.findByRole("dialog", { name: "Discard changes?" })).getByRole("button", { name: "Discard changes" }));
    expect(await screen.findByText("chat page")).toBeInTheDocument();
  });

  it("leaves without asking when nothing changed", async () => {
    serve(grid());
    const user = userEvent.setup();
    renderPage();

    await screen.findByRole("heading", { name: "Who's in each project" });
    await user.click(screen.getByRole("link", { name: "Leave" }));
    expect(await screen.findByText("chat page")).toBeInTheDocument();
  });

  it("is the Access page once set up, each tab saving from its own footer", async () => {
    serve(grid({ setup: { finished: true, finished_by_login: "alice", finished_at: new Date().toISOString() } }));
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByRole("heading", { name: "Access" })).toBeInTheDocument();
    expect(screen.getByText("2 projects · 1 public channel · 2 roles")).toBeInTheDocument();
    expect(screen.getByText("Set up")).toBeInTheDocument();
    expect(screen.getByText(/by alice, just now/)).toBeInTheDocument();
    await user.click(screen.getByRole("checkbox", { name: "Client in Storefront" }));
    await user.click(screen.getByRole("checkbox", { name: "Client in Marketing site" }));
    expect(screen.getByText("2 changes")).toBeInTheDocument();
    await user.click(screen.getByRole("checkbox", { name: "Client in Marketing site" }));
    expect(screen.getByText("1 change")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/workspaces/ws-1/access", { projects: [{ project_id: "p-sto", role_ids: ["r-dev", "r-client"], user_ids: [] }] });

    await user.click(screen.getByRole("tab", { name: "Public channels" }));
    await user.click(await screen.findByRole("checkbox", { name: "Developer in backend" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(mocks.put).toHaveBeenLastCalledWith("/api/workspaces/ws-1/access", { channels: [{ channel_id: "c-backend", hidden_role_ids: ["r-dev"] }] });
  });

  it("shows a manager only the Projects grid, and filters a long list", async () => {
    const projects = Array.from({ length: 12 }, (_, i) => ({ id: `p-${i}`, name: `Project ${i}`, prefix: `P${i}`, role_ids: ["r-dev"], user_ids: [] }));
    serve(grid({ setup: { finished: true }, projects, channels: [] }), false, ["projects:access"]);
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("12 projects · 2 roles")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Public channels" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Roles" })).not.toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: "Filter projects" }), "Project 1");
    expect(screen.getByText("Every project shown")).toBeInTheDocument();
    expect(screen.getAllByRole("checkbox", { name: /^Developer in/ })).toHaveLength(3);
    await user.clear(screen.getByRole("textbox", { name: "Filter projects" }));
    await user.type(screen.getByRole("textbox", { name: "Filter projects" }), "nothing");
    expect(screen.getByText(/No project matches/)).toBeInTheDocument();
  });

  it("puts Roles between the grids for a role editor, at its own path", async () => {
    serve(grid({ setup: { finished: true } }));
    const user = userEvent.setup();
    renderPage("/acme/access/roles");

    expect(await screen.findByText("Roles card")).toBeInTheDocument();
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Projects", "Roles", "Public channels"]);
    await user.click(screen.getByRole("tab", { name: "Projects" }));
    expect(await screen.findByRole("checkbox", { name: "Client in Storefront" })).toBeInTheDocument();
  });

  it("keeps Roles reachable for the Owner during the one-time setup", async () => {
    serve(grid());
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("tab", { name: "Roles" }));
    expect(await screen.findByText("Roles card")).toBeInTheDocument();
  });

  it("shows a role editor who manages no project the Roles card alone, without asking for the grids", async () => {
    serve(grid({ setup: { finished: true } }), false, ["roles:write"]);
    renderPage();

    expect(await screen.findByText("Roles card")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Access" })).toBeInTheDocument();
    expect(await screen.findByText("2 roles")).toBeInTheDocument();
    expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/workspaces/ws-1/access");
  });

  it("says what failed to load", async () => {
    serve(grid());
    const served = mocks.get.getMockImplementation();
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/access") throw new Error("down");
      return served?.(url);
    });
    renderPage();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });
});
