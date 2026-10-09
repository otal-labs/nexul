import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { WorkspaceSwitcher } from "@/components/WorkspaceSwitcher";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const baseUser = {
  id: "u1",
  provider: "github" as const,
  provider_user_id: "1",
  login: "onik97",
  name: "Onik",
  avatar_url: "",
  first_login_done: true,
  created_at: "2026-08-12T12:00:00Z",
};

const baseMe = { user: baseUser, needs_owner_wizard: false, needs_first_login_wizard: false, instance_permissions: [] as string[] };

const workspaces = [
  { id: "ws-1", name: "Globex", slug: "globex", created_at: "", updated_at: "" },
  { id: "ws-2", name: "Arena's Hub", slug: "arena-s-hub", created_at: "", updated_at: "" },
];

// GET /api/workspaces feeds the switcher's list; GET /api/auth/me carries workspaces:create in instance_permissions (useFetchMe).
const mockApi = (
  wsList: unknown,
  me: typeof baseMe = baseMe,
) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: me };
    if (url.endsWith("/me")) return { data: { role_name: "Member", permissions: ["tickets:read"] } };
    if (url === "/api/notifications/unread-count") return { data: { count: 3, workspaces: { "ws-2": 3 } } };
    return { data: wsList };
  });
};

const Location = () => <p data-testid="location">{useLocation().pathname}</p>;

const renderSwitcher = (collapsed = false, path = "/globex/board/ONLY") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <WorkspaceSwitcher collapsed={collapsed} />
        <Location />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("WorkspaceSwitcher", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    vi.mocked(api.post).mockReset();
    useSessionStore.setState({ token: "t", isLoggedIn: true });
    useWorkspaceStore.setState({ selectedWorkspaceId: "" });
    useWorkspaceStore.persist.clearStorage();
    localStorage.clear();
  });

  it("renders nothing while there are no workspaces", async () => {
    mockApi([]);
    renderSwitcher();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("shows the current workspace and lists every workspace with a checkmark on the selected one", async () => {
    // Selection repair lives in Layout now (useEnsureWorkspaceSelected), so the switcher is handed a selected id.
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    expect(await screen.findByText("Globex")).toBeInTheDocument();
    await user.click(screen.getByText("Globex"));

    const popover = screen.getByRole("dialog");
    const rows = within(popover).getAllByRole("button");
    const selectedRow = rows.find((row) => within(row).queryByText("Globex"));
    const otherRow = rows.find((row) => within(row).queryByText("Arena's Hub"));
    expect(selectedRow?.querySelector("svg.lucide-check")).not.toBeNull();
    expect(otherRow?.querySelector("svg.lucide-check")).toBeNull();
  });

  it("shows each workspace's unread inbox count, and none where the inbox is read", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Globex"));

    const popover = screen.getByRole("dialog");
    const rowOf = (name: string) => within(popover).getByText(name).closest("button")!;
    expect(await within(rowOf("Arena's Hub")).findByText("3")).toBeInTheDocument();
    expect(within(rowOf("Globex")).queryByText(/^\d+$/)).not.toBeInTheDocument();
  });

  it("keeps the section and drops the item when switching inside a workspace", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "globex" });
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Globex"));
    await user.click(screen.getByText("Arena's Hub"));

    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/arena-s-hub/board"));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("selects the workspace in place on a personal page, which has no workspace URL", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "globex" });
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher(false, "/settings/profile");

    await user.click(await screen.findByText("Globex"));
    await user.click(screen.getByText("Arena's Hub"));

    expect(useWorkspaceStore.getState()).toMatchObject({ selectedWorkspaceId: "ws-2", selectedWorkspaceSlug: "arena-s-hub" });
    expect(screen.getByTestId("location")).toHaveTextContent("/settings/profile");
  });

  it("keeps a valid persisted selection across refetches", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-2" });
    mockApi(workspaces);
    renderSwitcher();

    expect(await screen.findByText("Arena's Hub")).toBeInTheDocument();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
  });

  it("hides the New workspace row without workspaces:create", async () => {
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Globex"));
    expect(screen.queryByText("New workspace")).not.toBeInTheDocument();
  });

  it("shows the New workspace row with workspaces:create, and creates + switches on submit", async () => {
    mockApi(workspaces, { ...baseMe, instance_permissions: ["workspaces:create"] });
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "ws-3", name: "New Co", slug: "new-co", created_at: "", updated_at: "" },
    });
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Globex"));
    await user.click(screen.getByText("New workspace"));

    await user.type(screen.getByLabelText("Workspace name"), "New Co");
    await user.click(screen.getByRole("button", { name: "Create workspace" }));

    expect(api.post).toHaveBeenCalledWith("/api/workspaces", { name: "New Co" });
    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/new-co"));
  });

  it("renders an icon-only trigger when collapsed", async () => {
    mockApi(workspaces);
    renderSwitcher(true);

    await screen.findByRole("button", { name: "Globex" });
    expect(screen.queryByText("Globex")).not.toBeInTheDocument();
  });
});
