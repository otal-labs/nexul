import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
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

const catalog = [
  { value: "members:read", label: "Read members", domain: "members", action: "read", area: "workspace" },
  { value: "members:write", label: "Create and update members", domain: "members", action: "write", area: "workspace" },
  { value: "projects:read", label: "Read projects", domain: "projects", action: "read", area: "project" },
  { value: "projects:write", label: "Create and update projects", domain: "projects", action: "write", area: "project" },
  { value: "roles:read", label: "Read roles", domain: "roles", action: "read", area: "workspace" },
  { value: "roles:write", label: "Create and update roles", domain: "roles", action: "write", area: "workspace" },
  { value: "roles:delete", label: "Delete roles", domain: "roles", action: "delete", area: "workspace" },
  { value: "roles:clone", label: "Clone roles to another workspace", domain: "roles", action: "clone", area: "workspace" },
];

const level = (scope: HTMLElement, domain: string, name: string) =>
  within(within(scope).getByRole("radiogroup", { name: `${domain} access` })).getByRole("radio", { name });

const role = (overrides: Record<string, unknown> = {}) => ({
  id: "role-editor",
  workspace_id: "ws-1",
  name: "Editor",
  permissions: [],
  is_owner_role: false,
  created_at: "",
  updated_at: "",
  ...overrides,
});

const ownerRole = role({ id: "role-owner", name: "Owner", is_owner_role: true });

const openMenu = async (user: ReturnType<typeof userEvent.setup>, name: string, item: string) => {
  await user.click(await screen.findByRole("button", { name: `Actions for ${name}` }));
  await user.click(await screen.findByRole("button", { name: item }));
};

const holder = (roleId: string) => ({
  id: "u-bob",
  login: "bob",
  name: "Bob",
  avatar_url: "",
  status: "active",
  workspaces: [{ workspace_id: "ws-1", role_id: roleId }],
});

const mockRoles = (roles: unknown[], myPermissions: string[] = [], people: unknown[] = []) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/team") return Promise.resolve({ data: { people, workspaces: [], can_manage_accounts: false } });
    if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: roles });
    if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Admin", permissions: myPermissions } });
    if (url === "/api/workspaces") return Promise.resolve({ data: [] });
    if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <RoleSettingsSection />
    </QueryClientProvider>,
  );
};

describe("RoleSettingsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.delete.mockReset();
    mocks.errorMessage.mockClear();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("shows an error when the role list fails to load", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
      return Promise.reject(new Error("boom"));
    });
    mocks.errorMessage.mockReturnValue("Roles failed");
    renderSection();
    expect(await screen.findByText("Roles failed")).toBeInTheDocument();
  });

  it("shows the Owner role with no actions", async () => {
    mockRoles([ownerRole]);
    renderSection();

    expect(await screen.findByText("Owner")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Actions for Owner" })).not.toBeInTheDocument();
  });

  it("counts a role's areas per level and names the ones it can't open", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read", "members:write", "roles:read"] })]);
    renderSection();

    const row = (await screen.findByText("Editor")).closest("li")!;
    const levels = within(row).getByRole("list", { name: "Areas per level" });
    expect(levels).toHaveTextContent("Write1");
    expect(levels).toHaveTextContent("Read1");
    expect(levels).toHaveTextContent("None1");
    expect(within(row).getByText(/Projects/, { selector: "p" })).toHaveTextContent("No access · Projects");
  });

  it("creates a role with the chosen levels", async () => {
    mockRoles([ownerRole]);
    mocks.post.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.type(screen.getByLabelText("New role name"), "Editor");
    const form = screen.getByLabelText("New role name").closest("form")!;
    await user.click(level(form, "Every domain", "Read"));
    await user.click(level(form, "Projects", "Write"));
    await user.click(screen.getByRole("button", { name: /create role/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles", {
      name: "Editor",
      actions: ["members:read", "roles:read", "projects:read", "projects:write"],
    });
  });

  it("lists project areas under Every project and the rest under Workspace, from the catalog's area", async () => {
    mockRoles([ownerRole]);
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    const workspace = screen.getByRole("heading", { name: "Workspace" }).closest("section")!;
    const everyProject = screen.getByRole("heading", { name: "Every project" }).closest("section")!;
    expect(within(workspace).getByRole("radiogroup", { name: "Members access" })).toBeInTheDocument();
    expect(within(workspace).queryByRole("radiogroup", { name: "Projects access" })).not.toBeInTheDocument();
    expect(within(everyProject).getByRole("radiogroup", { name: "Every area access" })).toBeInTheDocument();
    expect(within(everyProject).getByRole("radiogroup", { name: "Projects access" })).toBeInTheDocument();
    expect(screen.getByText("Applies to members whose Every project is From role.")).toBeInTheDocument();
  });

  it("starts a new role from an existing role's permissions", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read"] })]);
    mocks.post.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.type(screen.getByLabelText("New role name"), "Auditor");
    await user.click(screen.getByRole("combobox", { name: "Copy permissions from role" }));
    await user.click(await screen.findByRole("option", { name: "Editor" }));
    await user.click(screen.getByRole("button", { name: /create role/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles", {
      name: "Auditor",
      actions: ["members:read"],
    });
  });

  it("does not call the api for an empty role name", async () => {
    mockRoles([ownerRole]);
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.click(screen.getByRole("button", { name: /create role/i }));
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("renames a role and changes its levels inline", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read", "members:write"] })]);
    mocks.patch.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    const input = screen.getByLabelText("Role name");
    const row = input.closest("li")!;
    await user.clear(input);
    await user.type(input, "Reviewer");
    await user.click(level(row, "Roles", "Write"));
    await user.click(screen.getByRole("button", { name: "Save role" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Reviewer",
      actions: ["members:read", "members:write", "roles:read", "roles:write"],
    });
  });

  it("deletes a role nobody holds after confirming", async () => {
    mockRoles([ownerRole, role()]);
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    renderSection();

    await screen.findAllByText("nobody");
    await openMenu(user, "Editor", "Delete");
    await user.click(await screen.findByRole("button", { name: "Delete role" }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor");
  });

  it("won't delete a role someone holds and says who to move first", async () => {
    mockRoles([ownerRole, role()], [], [holder("role-editor")]);
    const user = userEvent.setup();
    renderSection();

    await screen.findByText("1 person");
    await openMenu(user, "Editor", "Delete");
    expect(await screen.findByText("Editor is in use")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Got it" }));
    expect(mocks.delete).not.toHaveBeenCalled();
  });

  it("duplicates a role under a free copy name", async () => {
    mockRoles([ownerRole, role({ permissions: ["roles:read"] }), role({ id: "role-copy", name: "Editor (copy)" })]);
    mocks.post.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Duplicate");

    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles", { name: "Editor (copy 2)", actions: ["roles:read"] });
  });

  it("offers Clone to workspace only with roles:clone, and opens the dialog", async () => {
    mockRoles([ownerRole, role()], ["roles:write"]);
    const user = userEvent.setup();
    const { unmount } = renderSection();
    await user.click(await screen.findByRole("button", { name: "Actions for Editor" }));
    expect(screen.queryByRole("button", { name: "Clone to workspace…" })).not.toBeInTheDocument();
    unmount();

    mockRoles([ownerRole, role()], ["roles:write", "roles:clone"]);
    renderSection();
    await openMenu(user, "Editor", "Clone to workspace…");
    expect(await screen.findByRole("dialog", { name: "Clone Editor to another workspace" })).toBeInTheDocument();
  });

  it("toggles the Clone verb for Roles in the role editor", async () => {
    mockRoles([ownerRole, role({ permissions: ["roles:read"] })]);
    mocks.patch.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    await user.click(screen.getByRole("button", { name: "Clone roles to another workspace" }));
    await user.click(screen.getByRole("button", { name: "Save role" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Editor",
      actions: ["roles:read", "roles:clone"],
    });
  });
});
