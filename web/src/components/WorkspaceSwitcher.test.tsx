import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
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
  can_create_workspace: false,
  first_login_done: true,
  created_at: "2026-08-12T12:00:00Z",
};

const baseMe = { user: baseUser, needs_owner_wizard: false, needs_first_login_wizard: false };

const workspaces = [
  { id: "ws-1", name: "Shopkeepers", created_at: "", updated_at: "" },
  { id: "ws-2", name: "Arena's Hub", created_at: "", updated_at: "" },
];

// GET /api/workspaces feeds the switcher's list; GET /api/auth/me carries the can_create_workspace flag (useFetchMe).
const mockApi = (
  wsList: unknown,
  me: typeof baseMe = baseMe,
) => {
  vi.mocked(api.get).mockImplementation(async (url: string) =>
    url === "/api/auth/me" ? { data: me } : { data: wsList },
  );
};

const renderSwitcher = (collapsed = false) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <WorkspaceSwitcher collapsed={collapsed} />
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
    const { container } = renderSwitcher();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the current workspace and lists every workspace with a checkmark on the selected one", async () => {
    // Selection repair lives in Layout now (useEnsureWorkspaceSelected), so the switcher is handed a selected id.
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    expect(await screen.findByText("Shopkeepers")).toBeInTheDocument();
    await user.click(screen.getByText("Shopkeepers"));

    const popover = screen.getByRole("dialog");
    const rows = within(popover).getAllByRole("button");
    const selectedRow = rows.find((row) => within(row).queryByText("Shopkeepers"));
    const otherRow = rows.find((row) => within(row).queryByText("Arena's Hub"));
    expect(selectedRow?.querySelector("svg.lucide-check")).not.toBeNull();
    expect(otherRow?.querySelector("svg.lucide-check")).toBeNull();
  });

  it("selecting a different workspace updates the store and closes the popover", async () => {
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Shopkeepers"));
    await user.click(screen.getByText("Arena's Hub"));

    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(await screen.findByText("Arena's Hub")).toBeInTheDocument();
  });

  it("keeps a valid persisted selection across refetches", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-2" });
    mockApi(workspaces);
    renderSwitcher();

    expect(await screen.findByText("Arena's Hub")).toBeInTheDocument();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
  });

  it("hides the New Workspace row when can_create_workspace is false", async () => {
    mockApi(workspaces);
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Shopkeepers"));
    expect(screen.queryByText("New Workspace")).not.toBeInTheDocument();
  });

  it("shows the New Workspace row when can_create_workspace is true, and creates + switches on submit", async () => {
    mockApi(workspaces, { ...baseMe, user: { ...baseUser, can_create_workspace: true } });
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "ws-3", name: "New Co", created_at: "", updated_at: "" },
    });
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(await screen.findByText("Shopkeepers"));
    await user.click(screen.getByText("New Workspace"));

    await user.type(screen.getByLabelText("Workspace name"), "New Co");
    await user.click(screen.getByRole("button", { name: "Create workspace" }));

    expect(api.post).toHaveBeenCalledWith("/api/workspaces", { name: "New Co" });
    await waitFor(() => expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-3"));
  });

  it("renders an icon-only trigger when collapsed", async () => {
    mockApi(workspaces);
    renderSwitcher(true);

    await screen.findByTitle("Shopkeepers");
    expect(screen.queryByText("Shopkeepers")).not.toBeInTheDocument();
  });
});
