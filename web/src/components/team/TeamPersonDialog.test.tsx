import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TeamSection } from "@/components/team/TeamSection";
import type { Team, TeamMembership } from "@/models/Team";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "You can't give a level you don't hold" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const catalog = [
  { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
  { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "project" },
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read", area: "project" },
  { value: "chat:read", label: "Read chat", domain: "chat", action: "read", area: "workspace" },
];

const projects = [
  { id: "p-web", name: "Web", prefix: "WEB" },
  { id: "p-api", name: "Api", prefix: "API" },
];

const roles = (prefix: string) => [
  { id: `${prefix}-owner`, name: "Owner", is_owner: true },
  { id: `${prefix}-client`, name: "Client", is_owner: false },
  { id: `${prefix}-editor`, name: "Editor", is_owner: false },
];

const membership = (workspace: string, overrides: Partial<TeamMembership> = {}): TeamMembership => ({
  workspace_id: `ws-${workspace.toLowerCase()}`, workspace_name: workspace, role_id: `${workspace.toLowerCase()}-client`, role_name: "Client",
  is_owner: false, allow: [], deny: [], every_project: "role", projects: [], ...overrides,
});

const workspace = (name: string, canManage = true) => ({ id: `ws-${name.toLowerCase()}`, name, can_manage_members: canManage, roles: roles(name.toLowerCase()) });

const teamWith = (memberships: TeamMembership[], canManage = true): Team => ({
  can_manage_accounts: false,
  workspaces: [workspace("Nexul", canManage), workspace("Labs", canManage), workspace("Kit", canManage)],
  people: [{ id: "u-bob", login: "bob", name: "Bob", avatar_url: "", status: "active", created_at: "", online: false, last_seen_at: null, workspaces: memberships }],
});

const renderDialog = async (team: Team) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/me") return Promise.resolve({ data: { user: {}, instance_permissions: ["members:write"] } });
    if (url === "/api/team") return Promise.resolve({ data: team });
    if (url === "/api/projects") return Promise.resolve({ data: projects });
    if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
    return Promise.resolve({ data: [] });
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={["/configuration/team?person=u-bob"]}>
        <TeamSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return within(await screen.findByRole("dialog", { name: "Bob" }));
};

const pickRole = async (user: UserEvent, workspaceName: string, roleName: string) => {
  await user.click(screen.getByRole("combobox", { name: `Role in ${workspaceName}` }));
  await user.click(await screen.findByRole("option", { name: roleName }));
};

const level = (project: string, name: string) => within(screen.getByRole("radiogroup", { name: `${project} access` })).getByRole("radio", { name });

const memberPath = (workspaceId: string) => `/api/workspaces/${workspaceId}/members/u-bob`;

const webGrant = { project_id: "p-web", project_name: "Web", allow: ["tickets:read", "tickets:write"] };

describe("Team dialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("holds a change until Confirm, then sends it once and closes", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul"), membership("Labs")]));

    const confirm = dialog.getByRole("button", { name: "Confirm" });
    expect(confirm).toBeDisabled();
    await pickRole(user, "Nexul", "Editor");

    expect(mocks.patch).not.toHaveBeenCalled();
    expect(dialog.getByRole("tab", { name: "Nexul, unsaved changes" })).toBeInTheDocument();
    expect(dialog.getByRole("tab", { name: "Labs" })).toBeInTheDocument();
    await user.click(confirm);

    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Bob" })).not.toBeInTheDocument());
    expect(mocks.patch).toHaveBeenCalledTimes(1);
    expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), { role_id: "nexul-editor" });
  });

  it("asks before discarding on Cancel, and sends nothing once discarded", async () => {
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul")]));

    await pickRole(user, "Nexul", "Editor");
    await user.click(dialog.getByRole("button", { name: "Cancel" }));
    const prompt = within(await screen.findByRole("dialog", { name: "Discard changes?" }));
    await user.click(prompt.getByRole("button", { name: "Discard changes" }));

    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Bob" })).not.toBeInTheDocument());
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("adds a workspace from + as a selected tab, and applies adds, then changes, then removals", async () => {
    mocks.put.mockResolvedValue({});
    mocks.patch.mockResolvedValue({});
    mocks.delete.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul"), membership("Labs")]));

    await user.click(dialog.getByRole("button", { name: "Add to a workspace" }));
    await user.click(await screen.findByRole("button", { name: "Add" }));

    expect(dialog.getByRole("tab", { name: "Kit, unsaved changes" })).toHaveAttribute("aria-selected", "true");
    expect(mocks.put).not.toHaveBeenCalled();

    await user.click(dialog.getByRole("tab", { name: "Nexul" }));
    await pickRole(user, "Nexul", "Editor");
    await user.click(dialog.getByRole("tab", { name: "Labs" }));
    await user.click(dialog.getByRole("button", { name: "Remove from workspace" }));
    expect(mocks.delete).not.toHaveBeenCalled();
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    await vi.waitFor(() => expect(mocks.delete).toHaveBeenCalledWith(memberPath("ws-labs")));
    expect(mocks.put).toHaveBeenCalledWith(memberPath("ws-kit"), { role_id: "kit-client" });
    expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), { role_id: "nexul-editor" });
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
    await user.click(dialog.getByRole("tab", { name: "Nexul" }));
    await pickRole(user, "Nexul", "Editor");
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    expect(await dialog.findByRole("alert")).toHaveTextContent("Couldn't update Nexul: You can't give a level you don't hold");
    expect(dialog.getByRole("tab", { name: "Nexul, unsaved changes" })).toBeInTheDocument();

    await user.click(dialog.getByRole("button", { name: "Confirm" }));
    await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Bob" })).not.toBeInTheDocument());
    expect(mocks.put).toHaveBeenCalledTimes(1);
    expect(mocks.patch).toHaveBeenCalledTimes(2);
  });

  it("lists projects with the level buttons, held under From role and set once Chosen projects is picked", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul", { projects: [webGrant] })]));

    expect(await dialog.findByRole("radiogroup", { name: "Api access" })).toBeInTheDocument();
    expect(level("Api", "Read")).toBeDisabled();
    await user.click(dialog.getByRole("radio", { name: "Chosen projects" }));
    expect(level("Api", "Read")).toBeEnabled();
    expect(dialog.getByText("tickets Write")).toBeInTheDocument();
    await user.click(level("Api", "Read"));
    expect(mocks.patch).not.toHaveBeenCalled();
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    await vi.waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), {
        every_project: "none",
        project_access: [{ project_id: "p-api", allow: ["tickets:read", "docs:read"] }],
      }),
    );
  });

  it("opens a project's areas to give each its own level", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    const dialog = await renderDialog(teamWith([membership("Nexul", { every_project: "none", projects: [webGrant] })]));

    await user.click(await dialog.findByRole("button", { name: "Areas of Web" }));
    expect(level("Tickets", "Write")).toHaveAttribute("aria-checked", "true");
    await user.click(level("Docs", "Read"));
    await user.click(dialog.getByRole("button", { name: "Confirm" }));

    await vi.waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith(memberPath("ws-nexul"), {
        project_access: [{ project_id: "p-web", allow: ["tickets:read", "tickets:write", "docs:read"] }],
      }),
    );
  });

  it("shows a workspace the viewer can't manage as read only, with only the projects given", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", { every_project: "none", projects: [webGrant] })], false));

    expect(await dialog.findByText("Read only: you can't manage members in Nexul.")).toBeInTheDocument();
    expect(dialog.getByRole("radiogroup", { name: "Web access" })).toBeInTheDocument();
    expect(dialog.queryByRole("radiogroup", { name: "Api access" })).not.toBeInTheDocument();
    expect(dialog.getByRole("radio", { name: "Chosen projects" })).toBeDisabled();
    expect(dialog.queryByRole("combobox")).not.toBeInTheDocument();
    expect(dialog.queryByRole("button", { name: "Remove from workspace" })).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/projects", expect.anything());
  });

  it("gives the Owner no Every project row", async () => {
    const dialog = await renderDialog(teamWith([membership("Nexul", { role_id: "nexul-owner", role_name: "Owner", is_owner: true })]));

    expect(await dialog.findByText(/The Owner role can't be changed/)).toBeInTheDocument();
    expect(dialog.queryByRole("radiogroup", { name: /^Every project/ })).not.toBeInTheDocument();
  });
});
