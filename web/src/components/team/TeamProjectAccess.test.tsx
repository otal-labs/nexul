import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { toast } from "sonner";
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
  { id: "p-web", name: "Web storefront and its very long internal billing migration name", prefix: "WEB" },
  { id: "p-api", name: "Api", prefix: "API" },
];

const membership = (overrides: Partial<TeamMembership>): TeamMembership => ({
  workspace_id: "ws-nexul", workspace_name: "Nexul", role_id: "r-client", role_name: "Client", is_owner: false,
  allow: [], deny: [], every_project: "none", projects: [], ...overrides,
});

const teamWith = (workspaces: TeamMembership[], canManage = true): Team => ({
  can_manage_accounts: false,
  workspaces: [
    { id: "ws-nexul", name: "Nexul", can_manage_members: canManage, roles: [{ id: "r-owner", name: "Owner", is_owner: true }, { id: "r-client", name: "Client", is_owner: false }] },
  ],
  people: [{ id: "u-fahad", login: "fahad", name: "Fahad", avatar_url: "", status: "active", created_at: "", online: false, last_seen_at: null, workspaces }],
});

const renderTeam = async (team: Team) => {
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
      <MemoryRouter initialEntries={["/configuration/team?person=u-fahad"]}>
        <TeamSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return within(await screen.findByRole("dialog", { name: "Fahad" }));
};

const webGrant = { project_id: "p-web", project_name: projects[0]!.name, allow: ["tickets:read", "tickets:write"] };

describe("Team dialog, Project access", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("toasts the refusal and keeps the stored level when the server refuses a project's level", async () => {
    mocks.patch.mockRejectedValue(new Error("forbidden"));
    const user = userEvent.setup();
    await renderTeam(teamWith([membership({ projects: [webGrant] })]));

    await user.click(await screen.findByRole("button", { name: "Api access: None" }));
    await user.click(await screen.findByRole("menuitemradio", { name: /^Read/ }));

    await vi.waitFor(() => expect(toast.error).toHaveBeenCalledWith("You can't give a level you don't hold"));
    expect(screen.getByRole("button", { name: "Api access: None" })).toBeInTheDocument();
  });

  it("switches Every project to Only chosen projects, naming the person in the toast", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    await renderTeam(teamWith([membership({ every_project: "role" })]));

    expect(await screen.findByText("Every project, at their role's level.")).toBeInTheDocument();
    expect(screen.queryByText(/^Projects ·/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "Every project for Fahad" }));
    await user.click(await screen.findByRole("option", { name: "Only chosen projects" }));

    await vi.waitFor(() => expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-nexul/members/u-fahad", { every_project: "none" }));
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith("Fahad now sees only the Nexul projects you choose"));
  });

  it("counts the projects given, reads a mixed project as Custom with its summary, and sets a project's level by name", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    await renderTeam(teamWith([membership({ projects: [webGrant] })]));

    expect(await screen.findByText("Projects · 1 of 2")).toBeInTheDocument();
    expect(screen.getByText("Sees only the projects below. New projects stay hidden.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: `${projects[0]!.name} access: Custom` })).toBeInTheDocument();
    expect(screen.getByText("tickets Write")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Api access: None" }));
    await user.click(await screen.findByRole("menuitemradio", { name: /^Read/ }));

    await vi.waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-nexul/members/u-fahad", {
        project_access: [{ project_id: "p-api", allow: ["tickets:read", "docs:read"] }],
      }),
    );
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith("Fahad · Api: every area → Read"));
  });

  it("takes a project away with Remove access", async () => {
    mocks.patch.mockResolvedValue({});
    const user = userEvent.setup();
    await renderTeam(teamWith([membership({ projects: [webGrant] })]));

    await user.click(await screen.findByRole("button", { name: `${projects[0]!.name} access: Custom` }));
    await user.click(await screen.findByRole("menuitem", { name: "Remove access" }));

    await vi.waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-nexul/members/u-fahad", { project_access: [{ project_id: "p-web", allow: [] }] }),
    );
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(`Fahad no longer sees ${projects[0]!.name}`));
  });

  it("opens one project's areas at a time and closes them with Hide areas", async () => {
    const user = userEvent.setup();
    await renderTeam(teamWith([membership({ projects: [webGrant] })]));

    await user.click(await screen.findByRole("button", { name: "Api access: None" }));
    await user.click(await screen.findByRole("menuitem", { name: "Customize areas…" }));
    expect(screen.getByRole("button", { name: "Tickets access: None" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: `${projects[0]!.name} access: Custom` }));
    await user.click(await screen.findByRole("menuitem", { name: "Customize areas…" }));
    expect(screen.getByRole("button", { name: "Tickets access: Write" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Tickets access: None" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Hide areas" }));
    expect(screen.queryByRole("button", { name: /^Tickets access/ })).not.toBeInTheDocument();
  });

  it("shows a workspace the viewer can't manage as text: the Every project value and only the projects given", async () => {
    const dialog = await renderTeam(teamWith([membership({ projects: [webGrant] })], false));

    expect(await dialog.findByText("Only chosen projects")).toBeInTheDocument();
    expect(await dialog.findByText("Projects · 1 of 1")).toBeInTheDocument();
    expect(dialog.queryByRole("combobox")).not.toBeInTheDocument();
    expect(dialog.queryByRole("button", { name: /access:/ })).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/projects", expect.anything());
  });

  it("gives the Owner no Every project row", async () => {
    const dialog = await renderTeam(teamWith([membership({ role_id: "r-owner", role_name: "Owner", is_owner: true, every_project: "role" })]));

    expect(await dialog.findByText(/The Owner role can't be changed/)).toBeInTheDocument();
    expect(dialog.queryByText("Every project")).not.toBeInTheDocument();
  });
});
