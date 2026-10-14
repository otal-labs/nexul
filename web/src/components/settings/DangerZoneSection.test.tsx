import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "refused" }));
vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() }) }));

const workspaces = [
  { id: "workspace-default", slug: "home", name: "Home" },
  { id: "ws-acme", slug: "acme", name: "Acme" },
];

const person = (login: string) => ({ user_id: login, login, display_name: "", avatar_url: "" });

const renderSection = (workspaceId: string, held: string[]) => {
  useWorkspaceStore.setState({ selectedWorkspaceId: workspaceId });
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/workspaces") return Promise.resolve({ data: workspaces });
    if (url === `/api/workspaces/${workspaceId}/me`) return Promise.resolve({ data: { permissions: held, projects: [], roles: [] } });
    if (url.endsWith("/people")) return Promise.resolve({ data: { people: [person("alice"), person("bob")] } });
    if (url === "/api/projects") return Promise.resolve({ data: [{ id: "p-1" }, { id: "p-2" }, { id: "p-3" }] });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
  mocks.post.mockResolvedValue({ data: workspaces[1] });
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <DangerZoneSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("DangerZoneSection", () => {
  beforeEach(() => vi.clearAllMocks());

  it("names the people and projects that go in the archive confirmation, and archives only once confirmed", async () => {
    const user = userEvent.setup();
    renderSection("ws-acme", ["workspaces:delete"]);

    await user.click(await screen.findByRole("button", { name: "Archive workspace" }));
    expect(await screen.findByText("2 people lose this workspace.")).toBeInTheDocument();
    expect(screen.getByText(/3 projects, with their tickets, docs, chat and plays, go with it\./)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(mocks.post).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Archive workspace" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Archive workspace" }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-acme/archive"));
  });

  it.each([
    ["a member without workspaces:delete", "ws-acme", ["workspaces:write"]],
    ["the instance's own workspace, which can't be archived", "workspace-default", ["workspaces:delete"]],
  ])("offers no archive to %s", async (_name, workspaceId, held) => {
    renderSection(workspaceId, held);

    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith(`/api/workspaces/${workspaceId}/me`));
    expect(await screen.findByText(/To remove a project, open its settings/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Archive workspace" })).not.toBeInTheDocument();
  });
});
