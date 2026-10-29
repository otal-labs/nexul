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

const renderSection = (workspaceId: string, held: string[], showLocked = false) => {
  useWorkspaceStore.setState({ selectedWorkspaceId: workspaceId });
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/workspaces") return Promise.resolve({ data: workspaces });
    if (url === `/api/workspaces/${workspaceId}/me`) return Promise.resolve({ data: { permissions: held, projects: [], roles: [], show_locked_actions: showLocked } });
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

  it("keeps the archive row for the instance's own workspace, with the button off and the reason under it", async () => {
    renderSection("workspace-default", ["workspaces:delete"]);

    expect(await screen.findByRole("button", { name: "Archive workspace" })).toBeDisabled();
    expect(screen.getByText("The instance's own workspace can't be archived.")).toBeInTheDocument();
  });

  it("shows a member whose role shows locked actions the locked archive, and never archives", async () => {
    const user = userEvent.setup();
    renderSection("ws-acme", ["workspaces:write"], true);

    const locked = await screen.findByRole("button", { name: /Archive workspace/ });
    expect(locked).toHaveAttribute("aria-disabled", "true");
    expect(screen.queryByText("The instance's own workspace can't be archived.")).not.toBeInTheDocument();
    await user.click(locked);
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
