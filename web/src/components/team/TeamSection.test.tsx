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
  ],
  people: [
    {
      id: "u-bob", login: "bob", name: "Bob", avatar_url: "", status: "active", can_create_workspace: false, created_at: "",
      online: true, last_seen_at: null,
      workspaces: [
        { workspace_id: "ws-nexul", workspace_name: "Nexul", role_id: "r-editor", role_name: "Editor", is_owner: false, allow: [], deny: [] },
        { workspace_id: "ws-acme", workspace_name: "Acme", role_id: "r-viewer", role_name: "Viewer", is_owner: false, allow: [], deny: [] },
      ],
    },
  ],
};

const renderSection = (route = "/configuration/team", data: Team = team) => {
  mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/team" ? data : [] }));
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

  it("shows when each person was last online, with a presence dot and their account status kept as a word", async () => {
    const hoursAgo = (hours: number) => new Date(Date.now() - hours * 3_600_000).toISOString();
    const person = (id: string, name: string, extra: Partial<TeamPerson>): TeamPerson => ({
      id, login: id, name, avatar_url: "", status: "active", can_create_workspace: false, created_at: "", online: false, last_seen_at: null, workspaces: [], ...extra,
    });
    renderSection("/configuration/team", {
      ...team,
      people: [
        person("u-ann", "Ann", { online: true, last_seen_at: hoursAgo(1) }),
        person("u-cy", "Cy", { last_seen_at: hoursAgo(2) }),
        person("u-dee", "Dee", {}),
        person("u-eve", "Eve", { status: "disabled", last_seen_at: hoursAgo(72) }),
      ],
    });

    const dotOf = (row: HTMLElement) => row.querySelector("span[aria-hidden].rounded-full");
    const ann = await screen.findByRole("button", { name: "Open Ann" });
    expect(ann).toHaveTextContent("Online");
    expect(ann).toHaveTextContent("active");
    expect(dotOf(ann)).toHaveClass("bg-success");

    const cy = screen.getByRole("button", { name: "Open Cy" });
    expect(cy).toHaveTextContent("Last seen 2h ago");
    expect(dotOf(cy)).toHaveClass("bg-muted-foreground");
    expect(screen.getByRole("button", { name: "Open Dee" })).toHaveTextContent("Signed out");

    const eve = screen.getByRole("button", { name: "Open Eve" });
    expect(eve).toHaveTextContent("Last seen 3d ago");
    expect(eve).toHaveTextContent("disabled");
    expect(dotOf(eve)).toHaveClass("bg-muted-foreground");
  });

  it("keeps a workspace the viewer cannot manage read-only, with the reason, and edits the one they can", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Open Bob" }));

    const acme = within(await screen.findByRole("listitem", { name: "Acme" }));
    expect(acme.getByText("Viewer")).toBeInTheDocument();
    expect(acme.getByText("Read only: you can't manage members in Acme.")).toBeInTheDocument();
    expect(acme.queryByRole("combobox")).not.toBeInTheDocument();
    expect(acme.queryByRole("button", { name: /remove from/i })).not.toBeInTheDocument();

    expect(screen.getByRole("button", { name: "Disable" })).toBeInTheDocument();
    const nexul = within(screen.getByRole("listitem", { name: "Nexul" }));
    expect(nexul.getByRole("combobox", { name: "Role in Nexul" })).toBeInTheDocument();
    expect(nexul.getByRole("button", { name: "Remove from Nexul" })).toBeInTheDocument();
  });

  it("offers account actions only to a viewer who administers the instance", async () => {
    renderSection("/configuration/team?person=u-bob", { ...team, can_manage_accounts: false });

    await screen.findByRole("listitem", { name: "Nexul" });
    expect(screen.queryByRole("button", { name: /disable|remove account/i })).not.toBeInTheDocument();
  });

  it("adds the person to a workspace they are not in with the chosen role", async () => {
    mocks.put.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderSection("/configuration/team?person=u-bob");

    const labs = within(await screen.findByRole("listitem", { name: "Labs" }));
    await user.click(labs.getByRole("button", { name: "Add" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/workspaces/ws-labs/members/u-bob", { role_id: "r-tester" });
  });
});
