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
  { value: "members:read", label: "Read members", domain: "members", action: "read" },
  { value: "members:write", label: "Create and update members", domain: "members", action: "write" },
  { value: "projects:read", label: "Read projects", domain: "projects", action: "read" },
  { value: "projects:write", label: "Create and update projects", domain: "projects", action: "write" },
  { value: "roles:read", label: "Read roles", domain: "roles", action: "read" },
  { value: "roles:write", label: "Create and update roles", domain: "roles", action: "write" },
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

  it("summarises a custom role as one chip per domain at its level", async () => {
    mockRoles([ownerRole, role({ permissions: ["members:read", "members:write", "roles:read"] })]);
    renderSection();

    const row = (await screen.findByText("Editor")).closest("li")!;
    expect(within(row).getByText("Members · Write")).toBeInTheDocument();
    expect(within(row).getByText("Roles · Read")).toBeInTheDocument();
  });

  it("shows 'No permissions' for a role with no permissions", async () => {
    mockRoles([ownerRole, role()]);
    renderSection();

    expect(await screen.findByText("No permissions")).toBeInTheDocument();
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

    await user.click(await screen.findByRole("button", { name: "Rename role Editor" }));
    const input = screen.getByLabelText("Role name");
    const row = input.closest("li")!;
    await user.clear(input);
    await user.type(input, "Reviewer");
    await user.click(level(row, "Roles", "Write"));
    await user.click(screen.getByRole("button", { name: /new role/i }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/roles/role-editor", {
      name: "Reviewer",
      actions: ["members:read", "members:write", "roles:read", "roles:write"],
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
