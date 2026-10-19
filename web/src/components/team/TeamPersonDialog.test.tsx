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

const memberPath = (workspaceId: string) => `/api/workspaces/${workspaceId}/members/u-sam`;

describe("Team dialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("holds a ticked role until Confirm, then sends the whole set once and closes", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul"), membership("Labs")]));

    const confirm = dialog.getByRole("button", { name: "Confirm" });
    expect(confirm).toBeDisabled();
    expect(await dialog.findByText("1 of 2")).toBeInTheDocument();
    await user.click(dialog.getByRole("checkbox", { name: /Editor/ }));

    expect(mocks.patch).not.toHaveBeenCalled();
    expect(dialog.getByText("2 of 2")).toBeInTheDocument();
    expect(dialog.getByRole("tab", { name: "Nexul, unsaved changes" })).toBeInTheDocument();
    await user.click(confirm);

    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Sam Lee" })).not.toBeInTheDocument());
    expect(mocks.patch).toHaveBeenCalledTimes(1);
    expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), { role_ids: ["nexul-client", "nexul-editor"] });
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
      "Get the phone appThey chose to skip it; it comes back if a role gives it againSkipped· 2d ago",
    ]);
    expect(wizard.getByText("Done: Profile, Connect GitHub")).toBeInTheDocument();
  });

  it("puts what the viewer can't give under Can't give with the reason, keeping the person's tick", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", ["client", "lead"])]));

    const lead = await dialog.findByRole("checkbox", { name: /Lead/ });
    expect(lead).toBeDisabled();
    expect(lead).toBeChecked();
    expect(dialog.getByText("Holds Deploys: Delete, which you don't hold.")).toBeInTheDocument();
    expect(dialog.getByRole("checkbox", { name: /Owner/ })).toBeDisabled();
    expect(dialog.getByText("The Owner role is never given or taken here.")).toBeInTheDocument();
    expect(dialog.getByRole("checkbox", { name: /Client/ })).toBeEnabled();
  });

  it("warns once the last role goes, and still lets the empty set be confirmed", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul")]));

    await user.click(await dialog.findByRole("checkbox", { name: /Client/ }));
    expect(dialog.getByText("With no roles, Sam sees no projects or plays in Nexul.")).toBeInTheDocument();
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    await vi.waitFor(() => expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), { role_ids: [] }));
  });

  it("asks before discarding on Cancel, and sends nothing once discarded", async () => {
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul")]));

    await user.click(await dialog.findByRole("checkbox", { name: /Editor/ }));
    await user.click(dialog.getByRole("button", { name: "Cancel" }));
    const prompt = within(await screen.findByRole("dialog", { name: "Discard changes?" }));
    await user.click(prompt.getByRole("button", { name: "Discard changes" }));

    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Sam Lee" })).not.toBeInTheDocument());
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("adds a workspace from + once a role is picked, and applies adds, then changes, then removals", async () => {
    mocks.put.mockResolvedValue({});
    mocks.patch.mockResolvedValue({});
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul"), membership("Labs")]));

    await user.click(dialog.getByRole("button", { name: "Add to a workspace" }));
    await user.click(await screen.findByRole("button", { name: "Add" }));

    expect(dialog.getByRole("tab", { name: "Kit, unsaved changes" })).toHaveAttribute("aria-selected", "true");
    expect(await dialog.findByText("Pick at least one role to add Sam to Kit.")).toBeInTheDocument();
    expect(dialog.getByRole("button", { name: "Confirm" })).toBeDisabled();
    await user.click(dialog.getByRole("checkbox", { name: /Client/ }));

    await user.click(dialog.getByRole("tab", { name: "Nexul" }));
    await user.click(await dialog.findByRole("checkbox", { name: /Editor/ }));
    await user.click(dialog.getByRole("tab", { name: "Labs" }));
    await user.click(dialog.getByRole("button", { name: "Remove from workspace" }));
    expect(mocks.delete).not.toHaveBeenCalled();
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    await vi.waitFor(() => expect(mocks.delete).toHaveBeenCalledWith(memberPath("ws-labs")));
    expect(mocks.put).toHaveBeenCalledWith(memberPath("ws-kit"), { role_ids: ["kit-client"] });
    expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), { role_ids: ["nexul-client", "nexul-editor"] });
    const order = [mocks.put, mocks.patch, mocks.delete].map((fn) => fn.mock.invocationCallOrder[0]!);
    expect(order).toEqual([...order].sort((a, b) => a - b));
  });

  it("keeps the dialog and what is left when a change fails, and never resends what already applied", async () => {
    mocks.put.mockResolvedValue({});
    mocks.patch.mockRejectedValueOnce(new Error("forbidden")).mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul")]));

    await user.click(dialog.getByRole("button", { name: "Add to a workspace" }));
    await user.click(await screen.findByRole("button", { name: "Add" }));
    await user.click(await dialog.findByRole("checkbox", { name: /Client/ }));
    await user.click(dialog.getByRole("tab", { name: "Nexul" }));
    await user.click(await dialog.findByRole("checkbox", { name: /Editor/ }));
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    expect(await dialog.findByRole("alert")).toHaveTextContent("Couldn't update Nexul: You can't give a role holding more than you hold");
    expect(dialog.getByRole("tab", { name: "Nexul, unsaved changes" })).toBeInTheDocument();

    await user.click(dialog.getByRole("button", { name: "Confirm" }));
    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Sam Lee" })).not.toBeInTheDocument());
    expect(mocks.put).toHaveBeenCalledTimes(1);
    expect(mocks.patch).toHaveBeenCalledTimes(2);
  });

  it("shows a workspace the viewer can't manage as its role chips, read only", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", ["client", "editor"])], false));

    expect(await dialog.findByText("Read only: you can't manage members in Nexul.")).toBeInTheDocument();
    expect(dialog.getByText("Client")).toBeInTheDocument();
    expect(dialog.getByText("Editor")).toBeInTheDocument();
    expect(dialog.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(dialog.queryByRole("button", { name: "Remove from workspace" })).not.toBeInTheDocument();
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
    await dialog.findByRole("checkbox", { name: /Client/ });
    expect(dialog.queryByRole("button", { name: /View as/ })).not.toBeInTheDocument();
  });

  it("offers nothing for a person holding the workspace's Owner role", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", ["owner"])]), viewer);
    await dialog.findByText(/The Owner role is never given or taken here/);
    expect(dialog.queryByRole("button", { name: /View as/ })).not.toBeInTheDocument();
  });

  it("offers nothing for yourself", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul")]), { ...viewer, id: "u-sam" });
    await dialog.findByRole("checkbox", { name: /Client/ });
    expect(dialog.queryByRole("button", { name: /View as/ })).not.toBeInTheDocument();
  });
});
