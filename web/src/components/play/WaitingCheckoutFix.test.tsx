import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { QueuedRunRow } from "@/components/play/QueuedRunRow";
import type { ProjectLink } from "@/models/Pairing";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// Alice's workstation has two clones of the project's app, so a run there can't tell which to use.
const workstation = {
  id: "c-desk",
  name: "Workstation",
  token_expires_at: "2099-01-01T00:00:00Z",
  facts: {
    t3: { state: "answering" },
    projects: [
      { id: "t3-app-1", title: "app", path: "/home/alice/app", repository: "github.com/acme/app" },
      { id: "t3-app-2", title: "app-copy", path: "/home/alice/app-copy", repository: "github.com/acme/app" },
    ],
  },
};

const serve = (link: ProjectLink | undefined) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: { id: "u-alice", login: "alice" } } };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: [] } };
    if (url === "/api/pairing/computers") return { data: { computers: [workstation] } };
    if (url === "/api/projects/p-1/repos") return { data: [{ connector_id: "github", full_name: "acme/app", role: "app" }] };
    if (url === "/api/pairing/projects") return { data: { links: link ? [link] : [] } };
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Web" }] };
    return { data: [] };
  });

const held = (why: "ambiguous_checkout" | "no_checkout") =>
  queueItem({ reason: "no_computer_ready", waiting_on: [{ computer_id: "c-desk", name: "Workstation", why, text: "Workstation can't take it" }] });

const renderRow = (item = held("ambiguous_checkout")) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ul>
          <QueuedRunRow item={item} onLeave={() => undefined} />
        </ul>
      </MemoryRouter>
    </QueryClientProvider>,
  );

const pickTheCopy = async () => {
  const user = userEvent.setup();
  await user.click(await screen.findByRole("combobox", { name: "Pick the checkout on Workstation" }));
  await user.click(await screen.findByRole("option", { name: /app-copy/ }));
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.put).mockResolvedValue({ data: {} });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
});

describe("a waiting run held on a checkout", () => {
  it("picks the checkout on the run itself and saves it on the project's link for that computer", async () => {
    serve({ project_id: "p-1", computers: ["c-desk"], checkouts: [], model: "opus" });
    renderRow();

    await pickTheCopy();

    await waitFor(() => expect(api.put).toHaveBeenCalled());
    expect(api.put).toHaveBeenCalledWith("/api/pairing/projects/p-1", expect.objectContaining({
      model: "opus",
      computers: ["c-desk"],
      checkouts: [{ computer_id: "c-desk", repository: "acme/app", harness_project_id: "t3-app-2" }],
    }));
    expect(screen.queryByRole("link", { name: "Your computers" }), "the fix is on the run, not a link away").not.toBeInTheDocument();
  });

  it("keeps the pick for a computer outside the project's own list, leaving the list as it was", async () => {
    serve({ project_id: "p-1", computers: ["c-laptop"], checkouts: [] });
    renderRow();

    await pickTheCopy();

    await waitFor(() => expect(api.put).toHaveBeenCalled());
    expect(api.put).toHaveBeenCalledWith("/api/pairing/projects/p-1", expect.objectContaining({
      computers: ["c-laptop"],
      checkouts: [{ computer_id: "c-desk", repository: "acme/app", harness_project_id: "t3-app-2" }],
    }));
  });

  it("offers adding the project to a computer that has none", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/auth/me") return { data: { user: { id: "u-alice", login: "alice" } } };
      if (url === "/api/pairing/computers") return { data: { computers: [{ ...workstation, facts: { t3: { state: "answering" }, projects: [] } }] } };
      if (url === "/api/projects/p-1/repos") return { data: [{ connector_id: "github", full_name: "acme/app", role: "app" }] };
      if (url === "/api/pairing/projects") return { data: { links: [] } };
      return { data: [] };
    });
    renderRow(held("no_checkout"));

    expect(await screen.findByRole("button", { name: "Add it to Workstation" })).toBeInTheDocument();
  });
});
