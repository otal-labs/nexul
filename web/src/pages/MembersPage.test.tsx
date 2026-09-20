import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MembersPage } from "@/pages/MembersPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, patch: mocks.patch, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const roles = [
  { id: "role-owner", workspace_id: "ws-1", name: "Owner", permissions: [], is_owner_role: true, created_at: "", updated_at: "" },
  { id: "role-editor", workspace_id: "ws-1", name: "Editor", permissions: [], is_owner_role: false, created_at: "", updated_at: "" },
  { id: "role-admin", workspace_id: "ws-1", name: "Admin", permissions: [], is_owner_role: false, created_at: "", updated_at: "" },
];

let members: { user_id: string; login: string; role_id: string }[] = [];
let invites: { workspace_id: string; login: string; role_id: string; invited_by: string; created_at: string }[] = [];

const mockRoster = () => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: roles });
    if (url === "/api/workspaces/ws-1/members") return Promise.resolve({ data: { members, invites } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
};

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MembersPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("MembersPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.delete.mockReset();
    mocks.errorMessage.mockClear();
    members = [];
    invites = [];
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("lists the roster and pending invites", async () => {
    members = [{ user_id: "u-alice", login: "alice", role_id: "role-owner" }];
    invites = [{ workspace_id: "ws-1", login: "carol", role_id: "role-editor", invited_by: "u-alice", created_at: "" }];
    mockRoster();
    renderPage();

    expect(await screen.findByText("@alice")).toBeInTheDocument();
    expect(screen.getByText("Owner")).toBeInTheDocument();
    expect(screen.getByText("carol")).toBeInTheDocument();
    expect(screen.getByText("Invite sent")).toBeInTheDocument();
  });

  it("shows an empty state with no members and no invites", async () => {
    mockRoster();
    renderPage();

    expect(await screen.findByText(/no members yet/i)).toBeInTheDocument();
  });

  it("prompts to create a role first when the workspace has none besides Owner", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: [roles[0]] });
      return Promise.resolve({ data: { members, invites } });
    });
    renderPage();

    expect(await screen.findByText(/no new roles/i)).toBeInTheDocument();
    expect(screen.getByText(/create one before you can invite anyone/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /create a role/i })).toHaveAttribute("href", "/settings?section=roles");
    expect(screen.queryByLabelText("GitHub username")).not.toBeInTheDocument();
  });

  it("queues an invite with the default assignable role, then submits it", async () => {
    mockRoster();
    mocks.post.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage();

    await user.type(await screen.findByLabelText("GitHub username"), "bob");
    await user.click(screen.getByRole("button", { name: /add/i }));

    expect(await screen.findByText(/will invite \(1\)/i)).toBeInTheDocument();
    expect(screen.getByText("bob")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /continue/i }));
    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/members", {
      login: "bob",
      role_id: "role-editor",
    });
  });

  it("rejects queuing a login already queued or already a member", async () => {
    members = [{ user_id: "u-bob", login: "bob", role_id: "role-editor" }];
    mockRoster();
    const user = userEvent.setup();
    renderPage();

    await user.type(await screen.findByLabelText("GitHub username"), "bob");
    await user.click(screen.getByRole("button", { name: /add/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/already/i);
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("removes a member via the roster's remove button", async () => {
    members = [{ user_id: "u-bob", login: "bob", role_id: "role-editor" }];
    mockRoster();
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Remove bob" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/members/u-bob");
  });

  it("withdraws a pending invite", async () => {
    invites = [{ workspace_id: "ws-1", login: "carol", role_id: "role-editor", invited_by: "u-alice", created_at: "" }];
    mockRoster();
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Withdraw invite for carol" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/invites/carol");
  });

  it("changes a member's role via the roster's role select, offering only non-Owner roles", async () => {
    members = [{ user_id: "u-bob", login: "bob", role_id: "role-editor" }];
    mockRoster();
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage();

    const select = await screen.findByRole("combobox", { name: "Role for bob" });
    expect(screen.queryByRole("option", { name: "Owner" })).not.toBeInTheDocument();

    await user.click(select);
    await user.click(await screen.findByRole("option", { name: "Admin" }));
    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/members/u-bob", { role_id: "role-admin" });
  });

  it("shows an error when the roster fails to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Members failed");
    renderPage();

    expect(await screen.findByText("Members failed")).toBeInTheDocument();
  });
});
