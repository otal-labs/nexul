import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/components/ui/tooltip";
import { ProjectPeopleSection } from "@/components/settings/ProjectPeopleSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Project } from "@/models/Project";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Storefront", workspace_id: "ws-1" } as Project;
const role = (id: string, name: string, owner = false) => ({ id, workspace_id: "ws-1", name, permissions: [], is_owner_role: owner, created_at: "", updated_at: "" });
const person = (login: string) => ({ user_id: login, login, display_name: "", avatar_url: "" });
const rule = (over: object) => ({ resource_type: "project", resource_id: "p-1", target: "person", user_id: "", allow: [], deny: [], created_at: "2026-10-01", updated_at: "", ...over });

const team = {
  people: ["bob", "lena", "sam"].map((login) => ({ id: login, login, name: login, status: "active", workspaces: [{ workspace_id: "ws-1", role_id: "r-client", role_name: "Client", roles: [{ id: "r-client", name: "Client", is_owner: false }] }] })),
};

// The viewer manages the project's access; readsTeam adds members:write, which opens Team.
const respond = (grants: object[], readsTeam = false) =>
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/me") return Promise.resolve({ data: { user: { id: "alice" }, instance_permissions: readsTeam ? ["members:write"] : [] } });
    if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Lead", permissions: readsTeam ? ["members:write"] : [], projects: [{ project_id: "p-1", actions: ["projects:access"] }] } });
    if (url === "/api/team") return Promise.resolve({ data: team });
    if (url === "/api/permissions?resource_type=project&resource_id=p-1") return Promise.resolve({ data: { grants } });
    if (url === "/api/workspaces/ws-1/roles") return Promise.resolve({ data: [role("r-owner", "Owner", true), role("r-dev", "Developer"), role("r-client", "Client")] });
    if (url === "/api/workspaces/ws-1/people") return Promise.resolve({ data: { people: ["bob", "lena", "sam"].map(person) } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });

const renderCard = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TooltipProvider>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ProjectPeopleSection project={project} />
      </TooltipProvider>
    </QueryClientProvider>,
  );

describe("ProjectPeopleSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
    mocks.put.mockResolvedValue({ data: null });
  });

  it("lists the roles let in, the people named, and the people kept out under their own heading", async () => {
    respond([
      rule({ target: "role", role_id: "r-dev", allow: ["projects:open"] }),
      rule({ user_id: "lena", allow: ["tickets:write"] }),
      rule({ user_id: "bob", deny: ["projects:open"] }),
    ]);
    renderCard();

    const roles = await screen.findByRole("region", { name: "Roles" });
    expect(within(roles).getByText("Developer")).toBeInTheDocument();
    expect(within(roles).getByText("the role's own permissions")).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "People" })).getByText("1 rule here")).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "Kept out" })).getByText("bob")).toBeInTheDocument();
  });

  it("counts a role's holders for a viewer who may read Team", async () => {
    respond([rule({ target: "role", role_id: "r-client", allow: ["projects:open"] })], true);
    renderCard();
    expect(await screen.findByText("3 people hold it · the role's own permissions")).toBeInTheDocument();
  });

  it("asks nothing of Team for a viewer who can't read it, and shows the role without a count", async () => {
    respond([rule({ target: "role", role_id: "r-client", allow: ["projects:open"] })]);
    renderCard();
    expect(await screen.findByText("the role's own permissions")).toBeInTheDocument();
    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.queryByText(/hold it/)).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/team");
  });

  it("says only Owners can open a project that lists nobody", async () => {
    respond([]);
    renderCard();
    expect(await screen.findByText("Nobody is in Storefront yet. Only Owners can open it.")).toBeInTheDocument();
  });

  it("lists a role by allowing In this project, and offers neither the Owner role nor one already listed", async () => {
    const user = userEvent.setup();
    respond([rule({ target: "role", role_id: "r-dev", allow: ["projects:open"] })]);
    renderCard();

    await user.click(await screen.findByRole("button", { name: "Add a role or person" }));
    const choices = within(await screen.findByRole("dialog"));
    expect(choices.queryByRole("button", { name: /Owner/ })).not.toBeInTheDocument();
    expect(choices.queryByRole("button", { name: /Developer/ })).not.toBeInTheDocument();
    await user.click(choices.getByRole("button", { name: /Client/ }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/permissions", { resource_type: "project", resource_ids: ["p-1"], role_ids: ["r-client"], actions: ["projects:open"], state: "allow" }),
    );
  });

  it("asks before taking someone off, then drops every rule they hold here", async () => {
    const user = userEvent.setup();
    respond([rule({ user_id: "lena", allow: ["projects:open", "tickets:write"] })]);
    renderCard();

    await user.click(await screen.findByRole("button", { name: /^Actions for / }));
    await user.click(await screen.findByRole("button", { name: "Remove" }));
    await user.click(await screen.findByRole("button", { name: "Remove" }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/permissions", {
        resource_type: "project", resource_ids: ["p-1"], user_ids: ["lena"], actions: ["projects:open", "tickets:write"], state: "fallback",
      }),
    );
  });
});
