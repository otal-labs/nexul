import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TeamSection } from "@/components/team/TeamSection";
import type { Team, TeamPerson } from "@/models/Team";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
const dialog = vi.hoisted(() => ({ open: vi.fn() }));
vi.mock("@/hooks/useFormDialog", () => ({ useFormDialog: () => ({ open: dialog.open }) }));

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
      id: "u-bob", login: "bob", name: "Bob", avatar_url: "", status: "active", created_at: "", providers: ["github"],
      online: true, last_seen_at: null,
      workspaces: [
        { workspace_id: "ws-nexul", workspace_name: "Nexul", role_id: "r-editor", role_name: "Editor", is_owner: false, allow: [], deny: [], every_project: "role", projects: [] },
        { workspace_id: "ws-acme", workspace_name: "Acme", role_id: "r-viewer", role_name: "Viewer", is_owner: false, allow: [], deny: [], every_project: "role", projects: [] },
      ],
    },
  ],
};

const accountsBits = ["accounts:read", "accounts:write", "accounts:delete"];

const renderSection = (route = "/settings/team", data: Team = team, anywhere: string[] = accountsBits, invitations: unknown = []) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/me") return Promise.resolve({ data: { user: {}, instance_permissions: anywhere } });
    if (url === "/api/invitations") return Promise.resolve({ data: invitations });
    return Promise.resolve({ data: url === "/api/team" ? data : [] });
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/settings/:section?/:tab?" element={<TeamSection />} />
        </Routes>
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
      id, login: id, name, avatar_url: "", status: "active", created_at: "", providers: [], online: false, last_seen_at: null, workspaces: [], ...extra,
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

  it("shows a person's roles as chips, three then a count, and says when there are none", async () => {
    const roles = ["Owner", "Lead", "Editor", "Client", "Tester"].map((name) => ({ id: `r-${name}`, name, is_owner: name === "Owner" }));
    const nexul = team.people[0]!.workspaces[0]!;
    renderSection("/settings/team", {
      ...team,
      people: [
        { ...team.people[0]!, id: "u-lena", name: "Lena", workspaces: [{ ...nexul, roles }] },
        { ...team.people[0]!, id: "u-sam", name: "Sam", workspaces: [{ ...nexul, roles: [] }] },
      ],
    });

    const lena = await screen.findByRole("button", { name: "Open Lena" });
    expect(within(lena).getByText("Owner")).toBeInTheDocument();
    expect(within(lena).getByText("Editor")).toBeInTheDocument();
    expect(within(lena).queryByText("Client")).not.toBeInTheDocument();
    expect(within(lena).getByText("+2")).toHaveAttribute("title", "Client, Tester");
    expect(within(screen.getByRole("button", { name: "Open Sam" })).getByText("No roles")).toBeInTheDocument();
  });

  it("shows each linked sign-in account on the row with its username, never the email Discord signs in with", async () => {
    renderSection("/settings/team", {
      ...team,
      people: [{ ...team.people[0]!, providers: ["github", "discord"], usernames: { github: "bob", discord: "bob_d" } }],
    });

    const bob = await screen.findByRole("button", { name: "Open Bob" });
    expect(within(bob).getByRole("img", { name: "GitHub" })).toBeInTheDocument();
    expect(within(bob).getByRole("img", { name: "Discord" })).toBeInTheDocument();
    expect(within(bob).getByText("@bob")).toBeInTheDocument();
    expect(within(bob).getByText("bob_d")).toBeInTheDocument();
  });

  it("opens the Invitations tab with the new link on top once a link is created, and copies it", async () => {
    dialog.open.mockImplementation(async (options: { form: { props: { onCreated: (value: unknown) => void } } }) => {
      options.form.props.onCreated({ id: "inv-1", invited_by: "owner", created_at: "2026-09-20T00:00:00Z", expires_at: "2026-09-27T00:00:00Z", grants: [], url: "https://nexul.example.com/invite#secret" });
      return { success: true, data: null };
    });
    const user = userEvent.setup();
    renderSection();
    expect(await screen.findByRole("tab", { name: "People", selected: true })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Invite" }));

    expect(await screen.findByRole("tab", { name: "Invitations", selected: true })).toBeInTheDocument();
    const panel = screen.getByRole("tabpanel");
    const link = panel.firstElementChild as HTMLElement;
    expect(link).toHaveAttribute("role", "status");
    expect(link).toHaveTextContent("https://nexul.example.com/invite#secret");
    await user.click(within(link).getByRole("button", { name: "Copy link" }));
    expect(await within(link).findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("lists the links accepted and the links ended in the last 30 days, each accepted one opening the person it let in", async () => {
    const daysAgo = (days: number) => new Date(Date.now() - days * 86_400_000).toISOString();
    const grants = [{ workspace_id: "ws-nexul", workspace_name: "Nexul", role_id: "r-editor", role_name: "Editor", roles: [{ id: "r-editor", name: "Editor" }] }];
    const alice: TeamPerson = { ...team.people[0]!, id: "u-alice", login: "alice", name: "Alice", workspaces: [] };
    const bob: TeamPerson = {
      ...team.people[0]!,
      welcome: { first_run_done: true, dot: true, in_a_project: true, steps: [{ key: "computer", state: "left", given: false }] },
    };
    const user = userEvent.setup();
    renderSection("/settings/team/invitations", { ...team, people: [bob, alice] }, accountsBits, {
      invitations: [],
      accepted: [{ id: "inv-1", invited_by: "u-alice", created_at: daysAgo(3), expires_at: daysAgo(-4), redeemed_at: daysAgo(2), redeemed_by: "u-bob", grants }],
      ended: [{ id: "inv-2", invited_by: "u-alice", created_at: daysAgo(5), expires_at: daysAgo(4), grants }],
    });

    const accepted = within(await screen.findByRole("region", { name: "Accepted in the last 30 days" }));
    const row = await accepted.findByRole("button", { name: "Open Bob" });
    expect(row).toHaveTextContent("Editor in Nexul · invited by Alice");
    expect(row).toHaveTextContent("1 of 2 set up· computer not added");
    expect(row).toHaveTextContent("accepted 2d ago");
    const ended = within(screen.getByRole("region", { name: "Expired or deleted in the last 30 days" }));
    expect(ended.getByRole("listitem")).toHaveTextContent("Nexul · Editorby Alice · made 5d agoended 4d ago");

    await user.click(row);
    expect(await screen.findByRole("dialog", { name: "Bob" })).toBeInTheDocument();
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
    expect(acme.queryByText("Everything these roles allow, combined.")).not.toBeInTheDocument();
    expect(acme.queryByRole("button", { name: "Remove from workspace" })).not.toBeInTheDocument();

    await user.click(dialog.getByRole("tab", { name: "Nexul" }));
    const nexul = within(dialog.getByRole("tabpanel", { name: "Nexul" }));
    expect(nexul.getByText("Everything these roles allow, combined.")).toBeInTheDocument();
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
