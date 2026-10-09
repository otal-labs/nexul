import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TeamSection } from "@/components/team/TeamSection";
import type { Team, TeamPerson } from "@/models/Team";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const team: Team = {
  can_manage_accounts: true,
  workspaces: [
    { id: "ws-nexul", name: "Nexul", can_manage_members: true, roles: [{ id: "r-owner", name: "Owner", is_owner: true }, { id: "r-editor", name: "Editor", is_owner: false }] },
    { id: "ws-acme", name: "Acme", can_manage_members: false, roles: [{ id: "r-acme-owner", name: "Owner", is_owner: true }, { id: "r-viewer", name: "Viewer", is_owner: false }] },
    { id: "ws-labs", name: "Labs", can_manage_members: true, roles: [{ id: "r-labs-owner", name: "Owner", is_owner: true }, { id: "r-tester", name: "Tester", is_owner: false }] },
    { id: "ws-kit", name: "Kit", can_manage_members: true, roles: [{ id: "r-kit-owner", name: "Owner", is_owner: true }, { id: "r-guest", name: "Guest", is_owner: false }] },
    { id: "ws-ops", name: "Ops", can_manage_members: false, roles: [{ id: "r-ops-owner", name: "Owner", is_owner: true }, { id: "r-oncall", name: "On call", is_owner: false }] },
  ],
  people: [
    {
      id: "u-bob", login: "bob", name: "Bob", avatar_url: "", status: "active", created_at: "",
      online: true, last_seen_at: null,
      workspaces: [
        { workspace_id: "ws-nexul", workspace_name: "Nexul", role_id: "r-editor", role_name: "Editor", is_owner: false, allow: [], deny: [], every_project: "role", projects: [] },
        { workspace_id: "ws-acme", workspace_name: "Acme", role_id: "r-viewer", role_name: "Viewer", is_owner: false, allow: [], deny: [], every_project: "role", projects: [] },
      ],
    },
  ],
};

const accountsBits = ["accounts:read", "accounts:write", "accounts:delete"];

const renderSection = (route = "/settings/team", data: Team = team, anywhere: string[] = accountsBits) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/me") return Promise.resolve({ data: { user: {}, instance_permissions: anywhere } });
    return Promise.resolve({ data: url === "/api/team" ? data : [] });
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <TeamSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("TeamSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
  });

  it("shows when each person was last online with a presence dot, and names an account status only when it blocks sign-in", async () => {
    const hoursAgo = (hours: number) => new Date(Date.now() - hours * 3_600_000).toISOString();
    const person = (id: string, name: string, extra: Partial<TeamPerson>): TeamPerson => ({
      id, login: id, name, avatar_url: "", status: "active", created_at: "", online: false, last_seen_at: null, workspaces: [], ...extra,
    });
    renderSection("/settings/team", {
      ...team,
      people: [
        person("u-ann", "Ann", { online: true, last_seen_at: hoursAgo(1) }),
        person("u-cy", "Cy", { last_seen_at: hoursAgo(2) }),
        person("u-dee", "Dee", {}),
        person("u-eve", "Eve", { status: "disabled", last_seen_at: hoursAgo(72) }),
      ],
    });

    const dotOf = (row: HTMLElement) => row.querySelector("[data-presence]");
    const ann = await screen.findByRole("button", { name: "Open Ann" });
    expect(ann).toHaveTextContent("Online");
    expect(ann).not.toHaveTextContent(/active/i);
    expect(dotOf(ann)).toHaveClass("bg-success");

    const cy = screen.getByRole("button", { name: "Open Cy" });
    expect(cy).toHaveTextContent("Last seen 2h ago");
    expect(dotOf(cy)).toHaveClass("bg-muted-foreground");
    expect(screen.getByRole("button", { name: "Open Dee" })).toHaveTextContent("Signed out");

    const eve = screen.getByRole("button", { name: "Open Eve" });
    expect(eve).toHaveTextContent("Last seen 3d ago");
    expect(eve).toHaveTextContent("Disabled");
    expect(dotOf(eve)).toHaveClass("bg-muted-foreground");
  });

  it("gives each workspace the person is in a tab, read-only with the reason where the viewer can't manage members", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Open Bob" }));

    const dialog = within(await screen.findByRole("dialog", { name: "Bob" }));
    expect(dialog.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Nexul", "Acme"]);

    await user.click(dialog.getByRole("tab", { name: "Acme" }));
    const acme = within(dialog.getByRole("tabpanel", { name: "Acme" }));
    expect(acme.getByText("Viewer")).toBeInTheDocument();
    expect(acme.getByText("Read only: you can't manage members in Acme.")).toBeInTheDocument();
    expect(acme.queryByRole("combobox")).not.toBeInTheDocument();
    expect(acme.queryByRole("button", { name: "Remove from workspace" })).not.toBeInTheDocument();

    await user.click(dialog.getByRole("tab", { name: "Nexul" }));
    const nexul = within(dialog.getByRole("tabpanel", { name: "Nexul" }));
    expect(nexul.getByRole("combobox", { name: "Role in Nexul" })).toBeInTheDocument();
    expect(nexul.getByRole("button", { name: "Remove from workspace" })).toBeInTheDocument();
    expect(await dialog.findByRole("button", { name: "Disable" })).toBeInTheDocument();
  });

  // Changing an account's status takes accounts:write and removing it accounts:delete, each held in any workspace.
  it.each([
    { anywhere: [], disable: false, remove: false },
    { anywhere: ["accounts:write"], disable: true, remove: false },
    { anywhere: ["accounts:write", "accounts:delete"], disable: true, remove: true },
  ])("with $anywhere offers Disable: $disable, Remove account: $remove", async ({ anywhere, disable, remove }) => {
    renderSection("/settings/team?person=u-bob", team, anywhere);

    await screen.findByRole("tab", { name: "Nexul" });
    await vi.waitFor(() => expect(!!screen.queryByRole("button", { name: "Disable" })).toBe(disable));
    expect(!!screen.queryByRole("button", { name: "Remove account" })).toBe(remove);
  });

  it("offers + only the workspaces the person is not in and the viewer manages", async () => {
    const user = userEvent.setup();
    renderSection("/settings/team?person=u-bob");

    const dialog = within(await screen.findByRole("dialog", { name: "Bob" }));
    await user.click(dialog.getByRole("button", { name: "Add to a workspace" }));
    await user.click(await screen.findByRole("combobox", { name: "Workspace to add to" }));
    const offered = (await screen.findAllByRole("option")).map((option) => option.textContent);
    expect(offered).toEqual(["Labs", "Kit"]);
  });
});
