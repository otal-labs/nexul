import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MemoriesPage } from "@/pages/MemoriesPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: (error: unknown) => (error as Error)?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// The editor pane is MemoryPage's own concern; here it only has to be the open memory's.
vi.mock("@/pages/MemoryPage", async () => {
  const { useParams } = await import("react-router");
  return { MemoryPage: () => <p>editing {useParams().memoryId}</p> };
});

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const memory = (id: string, title: string, alwaysIncluded: boolean) => ({
  id,
  workspace_id: "ws-1",
  project_id: "p-1",
  kind: "",
  title,
  when_to_use: "",
  body: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"${title} body"}]}]}`,
  always_included: alwaysIncluded,
  version: 1,
  created_by: "u-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "u-1",
  updated_at: "2026-09-16T12:00:00Z",
});

const memories = [memory("mem-1", "Deploy quirks", true), memory("mem-2", "Naming rules", false)];

const renderPage = (path: string, permissions: string[], overrides: Record<string, unknown> = {}) => {
  const endpoints: Record<string, unknown> = {
    "/api/projects": [project],
    "/api/workspaces/ws-1/me": { role_name: "Member", permissions },
    "/api/memories": memories,
    ...overrides,
  };
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    const data = endpoints[url];
    if (data instanceof Error) throw data;
    return { data: data ?? [] };
  });
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/memories" element={<MemoriesPage />} />
          <Route path="/memories/:projectToken/:memoryId" element={<MemoriesPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const rowSwitch = (title: string) => screen.findByRole("switch", { name: `Always include ${title}` });

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
});

describe("MemoriesPage", () => {
  it("pins always-included memories first and opens the one the URL names", async () => {
    renderPage("/memories/BE/mem-2", ["memories:read"]);

    const pinned = await screen.findByRole("region", { name: "Pinned" });
    expect(within(pinned).getByRole("link", { name: /Deploy quirks/ })).toBeInTheDocument();
    expect(within(pinned).getByText("always in context")).toBeInTheDocument();
    expect(within(pinned).getByText("Deploy quirks body")).toBeInTheDocument();
    const other = screen.getByRole("region", { name: "Other" });
    expect(within(other).getByRole("link", { name: /Naming rules/ })).toHaveAttribute("aria-current", "page");
    expect(await screen.findByText("editing mem-2")).toBeInTheDocument();
  });

  it("the row switch saves always-included at once and rolls back when the save fails", async () => {
    const user = userEvent.setup();
    let failSave: (error: Error) => void = () => {};
    vi.mocked(api.put).mockReturnValue(new Promise((_, reject) => (failSave = reject)));
    renderPage("/memories", ["memories:read", "memories:write"]);

    const toggle = await rowSwitch("Naming rules");
    await vi.waitFor(() => expect(toggle).toBeEnabled());
    // Every later list read hangs, so the switch below shows the cache's own state, never a refetch's.
    const loaded = vi.mocked(api.get).getMockImplementation();
    vi.mocked(api.get).mockImplementation((url: string) =>
      url === "/api/memories" ? new Promise(() => {}) : (loaded?.(url) ?? Promise.resolve({ data: [] })),
    );
    await user.click(toggle);

    expect(await rowSwitch("Naming rules")).toBeChecked();
    expect(api.put).toHaveBeenCalledWith(
      "/api/memories/mem-2",
      expect.objectContaining({ title: "Naming rules", body: memories[1]?.body, always_included: true }),
    );
    failSave(new Error("conflict"));
    await vi.waitFor(async () => expect(await rowSwitch("Naming rules")).not.toBeChecked());
  });

  it("disables the row switch and leaves Clone out of the menu for a role without memories:write and memories:clone", async () => {
    const user = userEvent.setup();
    renderPage("/memories", ["memories:read", "memories:delete"]);

    // Delete's menu shows once the role has loaded, so what follows is the role's answer, not a pending fetch's.
    const menu = await screen.findByRole("button", { name: "More actions for Naming rules" });
    expect(await rowSwitch("Naming rules")).toBeDisabled();
    await user.click(menu);
    expect(await screen.findByRole("menuitem", { name: "Delete" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Clone" })).not.toBeInTheDocument();
  });

  it("offers Clone in the row menu to a role with memories:clone", async () => {
    const user = userEvent.setup();
    renderPage("/memories", ["memories:read", "memories:clone"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Naming rules" }));
    expect(await screen.findByRole("menuitem", { name: "Clone" })).toBeInTheDocument();
  });

  it("says so, and offers a writer New memory, when there are none", async () => {
    renderPage("/memories", ["memories:read", "memories:write"], { "/api/memories": [] });

    expect(await screen.findByText("No memories yet")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "New memory" })).toBeInTheDocument();
  });

  it("shows the shared error display when the list fails", async () => {
    renderPage("/memories", ["memories:read"], { "/api/memories": new Error("boom") });

    expect(await screen.findByText("Failed to load memories.")).toBeInTheDocument();
  });
});
