import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { toast } from "sonner";

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

// In the server's order: memories' template and projects' create are workspace verbs of project domains.
const catalog = [
  { value: "plays:read", label: "Read plays", domain: "plays", action: "read", area: "workspace" },
  { value: "plays:run", label: "Run plays", domain: "plays", action: "run", area: "workspace" },
  { value: "memories:read", label: "Read memories", domain: "memories", action: "read", area: "project" },
  { value: "memories:template", label: "Edit the workspace Interview template", domain: "memories", action: "template", area: "workspace" },
  { value: "members:read", label: "Read members", domain: "members", action: "read", area: "workspace" },
  { value: "members:write", label: "Create and update members", domain: "members", action: "write", area: "workspace" },
  { value: "projects:read", label: "Read projects", domain: "projects", action: "read", area: "project" },
  { value: "projects:write", label: "Create and update projects", domain: "projects", action: "write", area: "project" },
  { value: "projects:create", label: "Create projects", domain: "projects", action: "create", area: "workspace" },
  { value: "projects:access", label: "Manage this project's access", domain: "projects", action: "access", area: "project" },
  { value: "roles:read", label: "Read roles", domain: "roles", action: "read", area: "workspace" },
  { value: "roles:write", label: "Create and update roles", domain: "roles", action: "write", area: "workspace" },
  { value: "roles:delete", label: "Delete roles", domain: "roles", action: "delete", area: "workspace" },
  { value: "roles:clone", label: "Clone roles to another workspace", domain: "roles", action: "clone", area: "workspace" },
  { value: "runners:read", label: "Read runners", domain: "runners", action: "read", area: "instance" },
];

const level = (domain: string, name: string) =>
  within(within(screen.getByRole("dialog")).getByRole("radiogroup", { name: `${domain} access` })).getByRole("radio", { name });

const tab = (name: string) => screen.getByRole("tab", { name: new RegExp(`^${name}`) });

const ability = (name: string) => within(screen.getByRole("dialog")).getByRole("switch", { name: new RegExp(`^${name}`) });

const rowNames = () =>
  within(screen.getByRole("tabpanel"))
    .getAllByRole("listitem")
    .map((item) => item.textContent ?? "");

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
    expect(levels).toHaveTextContent("None4");
    expect(within(row).getByText(/Projects/, { selector: "p" })).toHaveTextContent("No access · Plays, Memories, Projects +1");
  });

  it("creates a role in the dialog, its levels set across the area tabs", async () => {
    mockRoles([ownerRole]);
    mocks.post.mockResolvedValue({ data: role({ name: "Support" }) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    expect(screen.getByRole("dialog", { name: "New role" })).toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: "Name" }), "Support");
    await user.click(tab("Workspace"));
    await user.click(level("All workspace areas", "Read"));
    await user.click(tab("In projects"));
    await user.click(level("Projects", "Write"));
    await user.click(screen.getByRole("button", { name: "Create role" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles", {
      name: "Support",
      actions: ["plays:read", "members:read", "roles:read", "projects:read", "projects:write"],
    });
    expect(toast.success).toHaveBeenCalledWith("Support created");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("puts each area under the tab for its catalog area, with a count of what's allowed", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read", "runners:read"] })]);
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    expect(tab("Instance")).toHaveAccessibleName("Instance, 1 of 1 allowed");
    expect(screen.getByRole("radiogroup", { name: "Runners access" })).toBeInTheDocument();
    expect(screen.getByText("Applies across every workspace on this install.")).toBeInTheDocument();
    await user.click(tab("Workspace"));
    expect(tab("Workspace")).toHaveAccessibleName("Workspace, 1 of 5 allowed");
    expect(screen.getByRole("radiogroup", { name: "Members access" })).toBeInTheDocument();
    expect(screen.queryByRole("radiogroup", { name: "Projects access" })).not.toBeInTheDocument();
    await user.click(tab("In projects"));
    expect(screen.getByRole("radiogroup", { name: "Projects access" })).toBeInTheDocument();
    expect(screen.getByText("Only inside the projects a holder is in. Each project lists who's in it.")).toBeInTheDocument();
    expect(screen.queryByText(/In this project|See this project/)).not.toBeInTheDocument();
  });

  it("leads each tab with its switches, which the bulk row never touches and the tally counts", async () => {
    mockRoles([ownerRole, role({ permissions: ["projects:create"] })]);
    mocks.patch.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    await user.click(tab("Workspace"));
    expect(rowNames().slice(1, 3)).toEqual([
      "Create projectsA new project lists its creator and the roles they pick.",
      "Edit the workspace Interview templateChange the questions every interview in this workspace asks.",
    ]);
    expect(ability("Create projects")).toBeChecked();
    expect(tab("Workspace")).toHaveAccessibleName("Workspace, 1 of 5 allowed");
    await user.click(level("All workspace areas", "None"));
    expect(ability("Create projects")).toBeChecked();
    await user.click(ability("Edit the workspace Interview template"));
    expect(tab("Workspace")).toHaveAccessibleName(/^Workspace, 2 of 5 allowed/);

    await user.click(tab("In projects"));
    expect(rowNames()[1]).toBe("Manage this project's accessChoose who's in a project and what they can do there.");
    expect(screen.queryByRole("checkbox", { name: "Manage this project's access" })).not.toBeInTheDocument();
    await user.click(ability("Manage this project's access"));
    await user.click(level("All areas in projects", "Read"));
    await user.click(level("Projects", "None"));
    expect(ability("Manage this project's access")).toBeChecked();
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Editor",
      actions: ["projects:create", "memories:template", "projects:access", "memories:read"],
    });
  });

  it("calls plays' first rung See, and their extras need See first", async () => {
    mockRoles([ownerRole, role()]);
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    await user.click(tab("Workspace"));
    expect(within(screen.getByRole("radiogroup", { name: "Plays access" })).getAllByRole("radio").map((rung) => rung.textContent)).toEqual([
      "None",
      "See",
    ]);
    expect(screen.getByRole("checkbox", { name: "Run plays" })).toBeDisabled();
    expect(screen.getByTitle("Needs See first")).toHaveTextContent("Run");
    expect(screen.getByTitle("Needs Read first")).toHaveTextContent("Clone");
  });

  it("starts a new role from a copy of another role's permissions", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read"] })]);
    mocks.post.mockResolvedValue({ data: role({ name: "Auditor" }) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.type(screen.getByRole("textbox", { name: "Name" }), "Auditor");
    await user.click(screen.getByRole("combobox", { name: "Start from" }));
    expect(screen.queryByRole("option", { name: "A copy of Owner" })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("option", { name: "A copy of Editor" }));
    expect(screen.getByText("Copied from Editor. Change anything below; Editor stays as it is.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Create role" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles", { name: "Auditor", actions: ["members:read"] });
  });

  it("keeps Create role off until the role has a name", async () => {
    mockRoles([ownerRole]);
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.click(level("All instance areas", "Read"));
    expect(screen.getByText("Name it to create it")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create role" })).toBeDisabled();
    await user.type(screen.getByRole("textbox", { name: "Name" }), "   ");
    expect(screen.getByRole("button", { name: "Create role" })).toBeDisabled();
  });

  it("renames a role and changes its levels, then saves them in one update", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read", "members:write"] })], [], [holder("role-editor")]);
    mocks.patch.mockResolvedValue({ data: role({ name: "Reviewer" }) });
    const user = userEvent.setup();
    renderSection();

    await screen.findByText("1 person");
    await openMenu(user, "Editor", "Edit");
    expect(screen.getByText("Held by Bob. Saving changes what they can do.")).toBeInTheDocument();
    expect(screen.getByText("No changes yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    const name = screen.getByRole("textbox", { name: "Name" });
    await user.clear(name);
    await user.type(name, "Reviewer");
    await user.click(tab("Workspace"));
    await user.click(level("Roles", "Write"));
    expect(tab("Workspace")).toHaveAccessibleName(/unsaved changes/);
    expect(tab("Instance")).not.toHaveAccessibleName(/unsaved changes/);
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Reviewer",
      actions: ["members:read", "members:write", "roles:read", "roles:write"],
    });
    expect(toast.success).toHaveBeenCalledWith("Reviewer saved");
  });

  it("asks before discarding changes, and closes at once with none", async () => {
    mockRoles([ownerRole, role()]);
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());

    await openMenu(user, "Editor", "Edit");
    await user.click(level("All instance areas", "Read"));
    await user.keyboard("{Escape}");
    expect(await screen.findByText("Your changes to Editor haven't been saved.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("dialog", { name: "Edit Editor" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close" }));
    await user.click(await screen.findByRole("button", { name: "Discard changes" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("opens and closes a new role without looping, and reopens it empty", async () => {
    mockRoles([ownerRole]);
    const errors = vi.spyOn(console, "error").mockImplementation(() => {});
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.type(screen.getByRole("textbox", { name: "Name" }), "Temp");
    await user.keyboard("{Escape}");
    expect(await screen.findByText("This role hasn't been created yet; what you set here will be lost.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Discard changes" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /new role/i }));

    expect(screen.getByRole("textbox", { name: "Name" })).toHaveValue("");
    expect(errors).not.toHaveBeenCalled();
    errors.mockRestore();
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

  it("deletes a role people hold after a confirm that says how many lose it", async () => {
    mockRoles([ownerRole, role()], [], [holder("role-editor"), { ...holder("role-editor"), id: "u-lena", login: "lena", name: "Lena" }]);
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    renderSection();

    await screen.findByText("2 people");
    await openMenu(user, "Editor", "Delete");
    expect(
      await screen.findByText("2 people lose this role. Anyone left with no role stays a member and sees no project. Deleting it can't be undone."),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete role" }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor");
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

  it("holds an area's extra verbs as checkboxes that need Read first", async () => {
    mockRoles([ownerRole, role({ permissions: ["roles:read"] })]);
    mocks.patch.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await openMenu(user, "Editor", "Edit");
    await user.click(tab("Workspace"));
    await user.click(screen.getByRole("checkbox", { name: "Clone roles to another workspace" }));
    await user.click(level("Roles", "None"));
    expect(screen.getByRole("checkbox", { name: "Clone roles to another workspace" })).toBeDisabled();
    expect(screen.getByRole("checkbox", { name: "Clone roles to another workspace" })).not.toBeChecked();
    await user.click(level("Roles", "Read"));
    await user.click(screen.getByRole("checkbox", { name: "Clone roles to another workspace" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Editor",
      actions: ["roles:read", "roles:clone"],
    });
  });
});
