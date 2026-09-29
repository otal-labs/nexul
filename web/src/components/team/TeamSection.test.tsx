import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TeamSection } from "@/components/team/TeamSection";
import type { Team } from "@/models/Team";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const team: Team = {
  workspaces: [
    { id: "ws-nexul", name: "Nexul", can_manage_members: true, roles: [{ id: "r-owner", name: "Owner", is_owner: true }, { id: "r-editor", name: "Editor", is_owner: false }] },
    { id: "ws-acme", name: "Acme", can_manage_members: false, roles: [{ id: "r-acme-owner", name: "Owner", is_owner: true }, { id: "r-viewer", name: "Viewer", is_owner: false }] },
    { id: "ws-labs", name: "Labs", can_manage_members: true, roles: [{ id: "r-labs-owner", name: "Owner", is_owner: true }, { id: "r-tester", name: "Tester", is_owner: false }] },
  ],
  people: [
    {
      id: "u-bob", login: "bob", name: "Bob", avatar_url: "", status: "active", can_create_workspace: false, created_at: "",
      workspaces: [
        { workspace_id: "ws-nexul", workspace_name: "Nexul", role_id: "r-editor", role_name: "Editor", is_owner: false, allow: [], deny: [] },
        { workspace_id: "ws-acme", workspace_name: "Acme", role_id: "r-viewer", role_name: "Viewer", is_owner: false, allow: [], deny: [] },
      ],
    },
  ],
};

const renderSection = (route = "/configuration/team") => {
  mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/team" ? team : [] }));
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

  it("lists each person with a one-line summary of their workspace access", async () => {
    renderSection();

    const row = await screen.findByRole("button", { name: "Open Bob" });
    expect(row).toHaveTextContent("Editor in Nexul · Viewer in Acme");
    expect(row).toHaveTextContent("active");
  });

  it("keeps a workspace the viewer cannot manage read-only, with the reason, and edits the one they can", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Open Bob" }));

    const acme = within(await screen.findByRole("listitem", { name: "Acme" }));
    expect(acme.getByText("Viewer")).toBeInTheDocument();
    expect(acme.getByText(/you need members:write in Acme/)).toBeInTheDocument();
    expect(acme.queryByRole("combobox")).not.toBeInTheDocument();
    expect(acme.queryByRole("button", { name: /remove from/i })).not.toBeInTheDocument();

    const nexul = within(screen.getByRole("listitem", { name: "Nexul" }));
    expect(nexul.getByRole("combobox", { name: "Role in Nexul" })).toBeInTheDocument();
    expect(nexul.getByRole("button", { name: "Remove from Nexul" })).toBeInTheDocument();
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
