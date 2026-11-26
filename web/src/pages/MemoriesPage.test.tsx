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

const memory = (id: string, title: string, alwaysIncluded: boolean, footer = false) => ({
  id,
  workspace_id: "ws-1",
  project_id: "p-1",
  kind: "",
  title,
  when_to_use: "",
  body: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"${title} body"}]}]}`,
  always_included: alwaysIncluded,
  footer,
  version: 1,
  created_by: "u-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "u-1",
  updated_at: "2026-09-16T12:00:00Z",
});

const memories = [
  memory("mem-1", "Deploy quirks", true),
  memory("mem-2", "Naming rules", false),
  memory("mem-3", "Where tickets go", false, true),
];

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
          <Route path="/acme/memories/:projectToken/:memoryId" element={<MemoriesPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

// Below 1024px the list is the page; a test that wants the side-by-side layout widens it.
const setWide = (wide: boolean) => {
  window.matchMedia = ((query: string) => ({ matches: wide, media: query, addEventListener: () => {}, removeEventListener: () => {} })) as unknown as typeof window.matchMedia;
};

beforeEach(() => {
  setWide(false);
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "", selectedProjectId: "" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
});

describe("MemoriesPage", () => {
  it("files memories into Main and Footer, required first, and opens the one the URL names", async () => {
    renderPage("/memories/BE/mem-2", ["memories:read"]);

    const main = await screen.findByRole("region", { name: "Main" });
    const [first, second] = within(main).getAllByRole("link");
    expect(first).toHaveAccessibleName(/Deploy quirks/);
    expect(first?.closest("li")).toHaveTextContent("Always included");
    expect(second?.closest("li")).not.toHaveTextContent("Always included");
    expect(second).toHaveAttribute("aria-current", "page");
    const footer = screen.getByRole("region", { name: "Footer" });
    expect(within(footer).getByRole("link", { name: /Where tickets go/ })).toBeInTheDocument();
    expect(within(footer).queryByRole("link", { name: /Naming rules/ })).not.toBeInTheDocument();
    expect(await screen.findByText("editing mem-2")).toBeInTheDocument();
  });

  it("opens the list's first memory from 1024px instead of an empty pane", async () => {
    setWide(true);
    useWorkspaceStore.setState({ selectedWorkspaceSlug: "acme" });
    renderPage("/memories", ["memories:read"]);
    expect(await screen.findByText("editing mem-1")).toBeInTheDocument();
  });

  it("moves a memory to the Footer folder from its row menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.put).mockResolvedValue({ data: {} });
    renderPage("/memories", ["memories:read", "memories:write"]);

    await user.pointer({ keys: "[MouseRight]", target: await screen.findByText("Naming rules") });
    await user.click(await screen.findByRole("menuitem", { name: "Move to" }));
    (await screen.findByRole("menuitemradio", { name: "Footer" })).focus();
    await user.keyboard("{Enter}");

    await vi.waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/memories/mem-2", expect.objectContaining({ title: "Naming rules", footer: true })),
    );
  });

  it("moves a row to the Footer at once and back when the save fails", async () => {
    const user = userEvent.setup();
    let failSave: (error: Error) => void = () => {};
    vi.mocked(api.put).mockReturnValue(new Promise((_, reject) => (failSave = reject)));
    renderPage("/memories", ["memories:read", "memories:write"]);

    await user.pointer({ keys: "[MouseRight]", target: await screen.findByText("Naming rules") });
    // Every later list read hangs, so the folders below show the cache's own state, never a refetch's.
    const loaded = vi.mocked(api.get).getMockImplementation();
    vi.mocked(api.get).mockImplementation((url: string) =>
      url === "/api/memories" ? new Promise(() => {}) : (loaded?.(url) ?? Promise.resolve({ data: [] })),
    );
    await user.click(await screen.findByRole("menuitem", { name: "Move to" }));
    (await screen.findByRole("menuitemradio", { name: "Footer" })).focus();
    await user.keyboard("{Enter}");

    const footer = screen.getByRole("region", { name: "Footer" });
    expect(await within(footer).findByRole("link", { name: /Naming rules/ })).toBeInTheDocument();
    failSave(new Error("conflict"));
    await vi.waitFor(() => expect(within(screen.getByRole("region", { name: "Main" })).getByRole("link", { name: /Naming rules/ })).toBeInTheDocument());
  });

  it("leaves Clone out of the menu for a role without memories:clone", async () => {
    const user = userEvent.setup();
    renderPage("/memories", ["memories:read", "memories:delete"]);

    await user.pointer({ keys: "[MouseRight]", target: await screen.findByText("Naming rules") });
    expect(await screen.findByRole("menuitem", { name: "Delete" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Clone" })).not.toBeInTheDocument();
  });

  it("offers Clone in the row menu to a role with memories:clone", async () => {
    const user = userEvent.setup();
    renderPage("/memories", ["memories:read", "memories:clone"]);

    await user.pointer({ keys: "[MouseRight]", target: await screen.findByText("Naming rules") });
    expect(await screen.findByRole("menuitem", { name: "Clone" })).toBeInTheDocument();
  });

  it("says so, and offers a writer New memory, when there are none", async () => {
    renderPage("/memories", ["memories:read", "memories:write"], { "/api/memories": [] });

    expect(await screen.findByText("No memories yet")).toBeInTheDocument();
    expect((await screen.findAllByRole("button", { name: "New memory" })).length).toBeGreaterThan(0);
  });

  it("shows the shared error display when the list fails", async () => {
    renderPage("/memories", ["memories:read"], { "/api/memories": new Error("boom") });

    expect(await screen.findByText("Couldn't load memories.")).toBeInTheDocument();
  });
});
