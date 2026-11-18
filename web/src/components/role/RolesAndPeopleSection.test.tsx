import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RolesAndPeopleSection } from "@/components/role/RolesAndPeopleSection";
import type { Role } from "@/models/Role";
import { AccountStatus, EveryProject, type Team, type TeamPerson } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn(), success: vi.fn(), error: vi.fn() }));
vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, patch: mocks.patch, delete: mocks.delete, post: vi.fn() },
  errorMessage: () => "refused",
}));
vi.mock("sonner", () => ({ toast: { success: mocks.success, error: mocks.error } }));

const LEAD = ["members:write", "members:view_as", "docs:read", "docs:write"];

const role = (id: string, name: string, permissions: string[], owner = false): Role => ({
  id,
  workspace_id: "ws-1",
  name,
  description: `${name} line`,
  permissions,
  welcome_steps: [],
  show_locked_actions: false,
  is_owner_role: owner,
  created_at: "",
  updated_at: "",
});

const roles = [
  role("r-owner", "Owner", [], true),
  role("r-lead", "Lead", LEAD),
  role("r-dev", "Developer", ["docs:read", "docs:write"]),
  role("r-admin", "Admin", ["docs:read", "docs:write", "docs:delete"]),
];

const catalog = ["read", "write", "delete"].map((action) => ({ value: `docs:${action}`, label: `${action} docs`, domain: "docs", action, area: "project" }));

const person = (id: string, name: string, roleIds: string[] | null): TeamPerson => ({
  id,
  login: id,
  name,
  avatar_url: "",
  status: AccountStatus.Active,
  created_at: "",
  providers: [],
  online: false,
  last_seen_at: null,
  workspaces: roleIds
    ? [
        {
          workspace_id: "ws-1",
          workspace_name: "Acme",
          role_id: roleIds[0] ?? "",
          role_name: "",
          roles: roleIds.map((id) => ({ id, name: roles.find((r) => r.id === id)!.name, is_owner: id === "r-owner" })),
          is_owner: roleIds.includes("r-owner"),
          allow: [],
          deny: [],
          every_project: EveryProject.Role,
          projects: [],
        },
      ]
    : [],
});

const team: Team = {
  people: [
    person("alice", "Alice Moreau", ["r-owner"]),
    person("bob", "Bob Okafor", ["r-dev", "r-lead"]),
    person("lena", "Lena Fischer", ["r-dev"]),
    person("kai", "Kai Nakamura", []),
    person("sam", "Sam Whitaker", null),
  ],
  workspaces: [{ id: "ws-1", name: "Acme", can_manage_members: true, roles: [] }],
  can_manage_accounts: false,
};

const serve = (permissions = LEAD) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/roles") return { data: roles };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Lead", roles: [{ id: "r-lead", name: "Lead", is_owner: false }], permissions } };
    if (url === "/api/workspaces") return { data: [{ id: "ws-1", name: "Acme", slug: "acme" }] };
    if (url === "/api/auth/me") return { data: { user: { id: "bob" }, instance_permissions: permissions } };
    if (url === "/api/permissions/catalog") return { data: { permissions: catalog } };
    if (url === "/api/team") return { data: team };
    if (url === "/api/invitations") return { data: { invitations: [], accepted: [], ended: [] } };
    return { data: [] };
  });

const renderAt = (path: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <RouterProvider router={createMemoryRouter([{ path: "/acme/access/roles/:roleId?", element: <RolesAndPeopleSection /> }], { initialEntries: [path] })} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  for (const mock of Object.values(mocks)) mock.mockReset();
  for (const write of [mocks.put, mocks.patch, mocks.delete]) write.mockResolvedValue({});
});

const holders = async (roleName: string) => within(await screen.findByRole("list", { name: `People holding ${roleName}` }));

describe("RolesAndPeopleSection", () => {
  it("names the person and the role when the server refuses a give, and keeps the role as it was", async () => {
    serve();
    mocks.patch.mockRejectedValue(new Error("nope"));
    const user = userEvent.setup();
    renderAt("/acme/access/roles/r-dev");

    await user.click(await screen.findByRole("button", { name: "Add people" }));
    await user.click(await screen.findByRole("button", { name: "Kai Nakamura" }));

    expect(mocks.error).toHaveBeenCalledWith("Couldn't give Developer to Kai Nakamura: refused");
    expect(mocks.success).not.toHaveBeenCalled();
  });

  it("gives a role to a member with the roles they hold kept, and adds someone outside the workspace with it", async () => {
    serve();
    const user = userEvent.setup();
    renderAt("/acme/access/roles/r-dev");

    await user.click(await screen.findByRole("button", { name: "Add people" }));
    const picker = within(await screen.findByRole("dialog"));
    expect(picker.queryByRole("button", { name: "Lena Fischer" })).not.toBeInTheDocument();
    expect(picker.queryByRole("button", { name: "Alice Moreau" })).not.toBeInTheDocument();
    await user.click(picker.getByRole("button", { name: "Kai Nakamura" }));
    await user.click(picker.getByRole("button", { name: "Sam Whitaker" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/members/kai", { role_ids: ["r-dev"] });
    expect(mocks.put).toHaveBeenCalledWith("/api/workspaces/ws-1/members/sam", { role_ids: ["r-dev"] });
    expect(mocks.success).toHaveBeenCalledWith("Developer given to Kai Nakamura");
    expect(mocks.success).toHaveBeenCalledWith("Sam Whitaker added to Acme as Developer");
  });

  it("takes the open role from a holder and leaves their other roles", async () => {
    serve();
    const user = userEvent.setup();
    renderAt("/acme/access/roles/r-dev");

    const list = await holders("Developer");
    await user.click(list.getByRole("button", { name: "Actions for Bob Okafor" }));
    await user.click(await screen.findByRole("button", { name: "Take Developer" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/members/bob", { role_ids: ["r-lead"] });
    expect(mocks.success).toHaveBeenCalledWith("Developer taken from Bob Okafor");
  });

  it("asks before taking someone out of the workspace, and sends nothing when cancelled", async () => {
    serve();
    const user = userEvent.setup();
    renderAt("/acme/access/roles/r-dev");

    const list = await holders("Developer");
    await user.click(list.getByRole("button", { name: "Actions for Lena Fischer" }));
    await user.click(await screen.findByRole("button", { name: "Remove from Acme" }));
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(mocks.delete).not.toHaveBeenCalled();

    await user.click(list.getByRole("button", { name: "Actions for Lena Fischer" }));
    await user.click(await screen.findByRole("button", { name: "Remove from Acme" }));
    await user.click(await screen.findByRole("button", { name: "Remove" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/members/lena");
  });

  it("offers no way to give a role holding more than the viewer holds, and says why", async () => {
    serve();
    renderAt("/acme/access/roles/r-admin");

    expect(await screen.findByText(/Holds .*Delete.*, which you don't hold\./)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add people" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Invite to Admin" })).not.toBeInTheDocument();
  });

  it("never gives or takes the Owner role, and leaves an Owner's row without a menu", async () => {
    serve();
    renderAt("/acme/access/roles/r-owner");

    const list = await holders("Owner");
    expect(list.getByText("Alice Moreau")).toBeInTheDocument();
    expect(list.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add people" })).not.toBeInTheDocument();
  });

  it("lists the members holding no role under No roles", async () => {
    serve();
    renderAt("/acme/access/roles/none");

    const list = within(await screen.findByRole("list", { name: "People holding no role" }));
    expect(list.getAllByRole("listitem").map((row) => row.textContent)).toEqual([expect.stringContaining("Kai Nakamura")]);
  });

  it("shows a role editor who manages no members what a role allows, and never asks for the Team", async () => {
    serve(["roles:write", "docs:read", "docs:write"]);
    renderAt("/acme/access/roles/r-dev");

    expect(await screen.findByRole("heading", { name: "Developer" })).toBeInTheDocument();
    expect(await screen.findByRole("region", { name: "In projects" })).toBeInTheDocument();
    expect(screen.queryByText(/^People/)).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /No roles/ })).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/team");
  });
});
