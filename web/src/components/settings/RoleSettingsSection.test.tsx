import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  { value: "members:write", label: "Manage members", domain: "members", action: "write" },
  { value: "projects:write", label: "Manage projects", domain: "projects", action: "write" },
  { value: "roles:write", label: "Manage roles", domain: "roles", action: "write" },
];

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

const mockRoles = (roles: unknown[]) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: roles });
    if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
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

  it("shows the Owner role with a badge and no edit/delete controls", async () => {
    mockRoles([ownerRole]);
    renderSection();

    expect(await screen.findByText("Owner")).toBeInTheDocument();
    expect(screen.getByText("Protected role")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /rename role owner/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete role owner/i })).not.toBeInTheDocument();
  });

  it("lists a custom role's permission chips with catalog labels", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:write"] })]);
    renderSection();

    const row = (await screen.findByText("Editor")).closest("li")!;
    expect(within(row).getByText("Manage members")).toBeInTheDocument();
  });

  it("shows 'No permissions' for a role with no permissions", async () => {
    mockRoles([ownerRole, role()]);
    renderSection();

    expect(await screen.findByText("No permissions")).toBeInTheDocument();
  });

  it("creates a role with the checked permissions", async () => {
    mockRoles([ownerRole]);
    mocks.post.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await user.type(await screen.findByLabelText("New role name"), "Editor");
    await user.click(screen.getByRole("checkbox", { name: "Manage projects" }));
    await user.click(screen.getByRole("button", { name: /create role/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/roles", {
      name: "Editor",
      actions: ["projects:write"],
    });
  });

  it("does not call the api for an empty role name", async () => {
    mockRoles([ownerRole]);
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /create role/i }));
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("renames a role and toggles its permissions inline", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:write"] })]);
    mocks.patch.mockResolvedValue({ data: role() });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Rename role Editor" }));
    const input = screen.getByLabelText("Role name");
    const row = within(input.closest("li")!);
    await user.clear(input);
    await user.type(input, "Reviewer");
    await user.click(row.getByRole("checkbox", { name: "Manage roles" }));
    await user.click(screen.getByLabelText("New role name"));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Reviewer",
      actions: ["members:write", "roles:write"],
    });
  });

  it("deletes a role after confirming", async () => {
    mockRoles([ownerRole, role()]);
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Delete role Editor" }));
    await user.click(screen.getByRole("button", { name: /^confirm$/i }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor");
  });
});
