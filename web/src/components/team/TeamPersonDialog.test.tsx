import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TeamSection } from "@/components/team/TeamSection";
import type { Team, TeamMembership } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "You can't give a role holding more than you hold" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
  { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "project" },
  { value: "deploys:read", label: "Read deploys", domain: "deploys", action: "read", area: "project" },
  { value: "deploys:write", label: "Start deploys", domain: "deploys", action: "write", area: "project" },
  { value: "deploys:delete", label: "Delete deploys", domain: "deploys", action: "delete", area: "project" },
];

// The viewer holds Editor everywhere, so Lead, which deletes deploys, is beyond them.
const roles = (ws: string) => [
  { id: `${ws}-owner`, name: "Owner", permissions: [], is_owner_role: true },
  { id: `${ws}-client`, name: "Client", permissions: ["tickets:read"], is_owner_role: false },
  { id: `${ws}-editor`, name: "Editor", permissions: ["tickets:read", "tickets:write", "deploys:read"], is_owner_role: false },
  { id: `${ws}-lead`, name: "Lead", permissions: ["tickets:read", "deploys:read", "deploys:write", "deploys:delete"], is_owner_role: false },
];

const teamRoles = (ws: string) => roles(ws).map((role) => ({ id: role.id, name: role.name, is_owner: role.is_owner_role }));

const membership = (workspace: string, held: string[] = ["client"]): TeamMembership => {
  const ws = workspace.toLowerCase();
  const named = teamRoles(ws).filter((role) => held.some((id) => role.id === `${ws}-${id}`));
  return {
    workspace_id: `ws-${ws}`, workspace_name: workspace, role_id: named[0]?.id ?? "", role_name: named.map((r) => r.name).join(", "), roles: named,
    is_owner: named.some((role) => role.is_owner), allow: [], deny: [], every_project: "role", projects: [],
  };
};

const workspace = (name: string, canManage = true) => ({ id: `ws-${name.toLowerCase()}`, name, can_manage_members: canManage, roles: teamRoles(name.toLowerCase()) });

const teamWith = (memberships: TeamMembership[], canManage = true): Team => ({
  can_manage_accounts: false,
  workspaces: [workspace("Nexul", canManage), workspace("Labs", canManage), workspace("Kit", canManage)],
  people: [{ id: "u-sam", login: "sam", name: "Sam Lee", avatar_url: "", status: "active", created_at: "", providers: ["github"], online: false, last_seen_at: null, workspaces: memberships }],
});

const renderDialog = async (team: Team, viewer: { id?: string; holds?: string[] } = {}) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: viewer.id }, instance_permissions: ["members:write"] } });
    if (url === "/api/team") return Promise.resolve({ data: team });
    if (url === "/api/workspaces") return Promise.resolve({ data: [{ id: "ws-nexul", name: "Nexul", slug: "nexul" }] });
    if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
    const rolesOf = /^\/api\/workspaces\/ws-(\w+)\/roles$/.exec(url);
    if (rolesOf) return Promise.resolve({ data: roles(rolesOf[1]!) });
    const meIn = /^\/api\/workspaces\/ws-(\w+)\/me$/.exec(url);
    if (meIn) return Promise.resolve({ data: { role_name: "Editor", roles: [{ id: `${meIn[1]}-editor`, name: "Editor", is_owner: false }], permissions: viewer.holds ?? [] } });
    return Promise.resolve({ data: [] });
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={["/configuration/team?person=u-sam"]}>
        <TeamSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return within(await screen.findByRole("dialog", { name: "Sam Lee" }));
};


describe("Team dialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows who invited the person, and their Welcome wizard: each step left or skipped as a row, the done ones folded", async () => {
    const base = teamWith([membership("Nexul")]);
    const twoDaysAgo = new Date(Date.now() - 2 * 86_400_000).toISOString();
    const alice = { ...base.people[0]!, id: "u-alice", login: "alice", name: "Alice" };
    const sam = {
      ...base.people[0]!,
      joined: { invited_by: "u-alice", accepted_at: twoDaysAgo },
      welcome: {
        first_run_done: true, dot: true, in_a_project: true,
        steps: [
          { key: "github" as const, state: "done" as const, given: false },
          { key: "computer" as const, state: "left" as const, given: false },
          { key: "phone" as const, state: "skipped" as const, at: twoDaysAgo, given: false },
        ],
      },
    };
    const dialog = await renderDialog({ ...base, people: [sam, alice] });

    expect(dialog.getByText("Joined 2d ago, invited by Alice")).toBeInTheDocument();
    const wizard = within(dialog.getByRole("region", { name: "Welcome wizard" }));
    expect(wizard.getByText("2 of 4 set up")).toBeInTheDocument();
    const rows = wizard.getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual([
      "Your computerNo paired computer with its setup confirmedNot done",
      "Get the phone appThey chose to skip it; it comes back if a role gives it againSkipped, 2d ago",
    ]);
    expect(wizard.getByText("Done: Profile, Connect GitHub")).toBeInTheDocument();
  });

  it("shows each workspace's roles as chips and changes none, linking to the Access page of a workspace the viewer is in and manages", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", ["client", "editor"]), membership("Labs")]));

    expect(await dialog.findByRole("link", { name: "Change on Access" })).toHaveAttribute("href", "/nexul/access/roles");
    expect(dialog.getByText("Editor")).toBeInTheDocument();
    expect(dialog.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(dialog.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    await userEvent.setup().click(dialog.getByRole("tab", { name: "Labs" }));
    expect(await dialog.findByText("Client")).toBeInTheDocument();
    expect(dialog.queryByRole("link", { name: "Change on Access" })).not.toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("shows an Owner's roles without a checklist", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", ["owner"])]));

    expect(await dialog.findByText(/The Owner role is never given or taken here/)).toBeInTheDocument();
    expect(dialog.queryByRole("checkbox")).not.toBeInTheDocument();
  });
});

describe("Team dialog, View as", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-nexul" });
  });

  const viewer = { id: "u-viewer", holds: ["members:view_as"] };

  it("offers View as on a member's workspace to a viewer who holds it", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul")]), viewer);
    expect(await dialog.findByRole("button", { name: "View as Sam" })).toBeEnabled();
  });

  it("offers nothing to a viewer without the permission", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul")]), { id: "u-viewer" });
    await dialog.findByText("Client");
    expect(dialog.queryByRole("button", { name: /View as/ })).not.toBeInTheDocument();
  });

  it("offers nothing for a person holding the workspace's Owner role", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", ["owner"])]), viewer);
    await dialog.findByText(/The Owner role is never given or taken here/);
    expect(dialog.queryByRole("button", { name: /View as/ })).not.toBeInTheDocument();
  });

  it("offers nothing for yourself", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul")]), { ...viewer, id: "u-sam" });
    await dialog.findByText("Client");
    expect(dialog.queryByRole("button", { name: /View as/ })).not.toBeInTheDocument();
  });
});
