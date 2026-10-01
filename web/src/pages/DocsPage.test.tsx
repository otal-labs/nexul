import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocsPage } from "@/pages/DocsPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// The editor pane is DocPage's own concern (collab session, tiptap); here it only has to be the open doc's.
vi.mock("@/pages/DocPage", async () => {
  const { useParams } = await import("react-router");
  return { DocPage: () => <p>editing {useParams().docId}</p> };
});

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const doc = (id: string, title: string, updated_at: string, can_open = true, locked = false) => ({
  id,
  project_id: "p-1",
  title,
  version: 1,
  archived: false,
  locked,
  can_open,
  created_at: updated_at,
  updated_at,
  ...(can_open ? { created_by: "u-1", snippet: `${title} notes` } : {}),
});

const docs = [
  doc("doc-1", "Storage Spine", new Date().toISOString(), true, true),
  doc("doc-2", "Rollback plan", "2020-01-01T12:00:00Z"),
  doc("doc-3", "Salaries", "2020-01-02T12:00:00Z", false),
];

const LocationSpy = () => <p data-testid="location">{useLocation().pathname}</p>;

const renderPage = (path: string, permissions: string[], overrides: Record<string, unknown> = {}) => {
  const endpoints: Record<string, unknown> = {
    "/api/projects": [project],
    "/api/workspaces": [{ id: "ws-1", name: "Acme", slug: "acme" }],
    "/api/workspaces/ws-1/me": { role_name: "Member", permissions },
    "/api/docs": docs,
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
          <Route path="/acme/docs" element={<DocsPage />} />
          <Route path="/acme/docs/:projectToken/:docId" element={<DocsPage />} />
          <Route path="/acme/docs/:docId" element={<DocsPage />} />
        </Routes>
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DocsPage", () => {
  it("lists the openable docs by day and opens the one the URL names", async () => {
    renderPage("/acme/docs/BE/doc-2", ["docs:read"]);

    const earlier = await screen.findByRole("region", { name: "Earlier" });
    expect(within(earlier).getByRole("link", { name: /Rollback plan/ })).toHaveAttribute("aria-current", "page");
    const today = screen.getByRole("region", { name: "Today" });
    expect(within(today).getByRole("link", { name: /Storage Spine/ })).not.toHaveAttribute("aria-current");
    expect(screen.queryByText("Salaries")).not.toBeInTheDocument();
    expect(await screen.findByText("editing doc-2")).toBeInTheDocument();
  });

  it("moves a bare doc link to its project's URL", async () => {
    renderPage("/acme/docs/doc-2", ["docs:read"], { "/api/docs/doc-2": { ...doc("doc-2", "Rollback plan", ""), body: "" } });

    await vi.waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/acme/docs/BE/doc-2"));
    expect(await screen.findByText("editing doc-2")).toBeInTheDocument();
  });

  it("shows each row as its title alone, marking a locked one", async () => {
    renderPage("/acme/docs", ["docs:read"]);

    const row = await screen.findByRole("link", { name: /Storage Spine/ });
    expect(row).toHaveTextContent(/^Storage Spine$/);
    expect(within(row).getByLabelText("Locked")).toBeInTheDocument();
    expect(within(screen.getByRole("link", { name: /Rollback plan/ })).queryByLabelText("Locked")).not.toBeInTheDocument();
  });

  it("offers New doc and each row action only to a role that holds them", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:delete"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Storage Spine" }));
    expect(await screen.findByRole("menuitem", { name: "Delete" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Unlock" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Clone" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New doc" })).not.toBeInTheDocument();
  });

  it("locks and unlocks a doc from its row menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    renderPage("/acme/docs", ["docs:read", "docs:write", "docs:clone", "docs:delete"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Rollback plan" }));
    const items = await screen.findAllByRole("menuitem");
    expect(items.map((item) => item.textContent)).toEqual(["Lock", "Clone", "Delete"]);
    await user.click(screen.getByRole("menuitem", { name: "Lock" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-2/lock"));

    await user.click(screen.getByRole("button", { name: "More actions for Storage Spine" }));
    await user.click(await screen.findByRole("menuitem", { name: "Unlock" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/unlock"));
  });

  it("clones a doc into the project picked in Clone to…", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "doc-9", project_id: "p-1" } });
    renderPage("/acme/docs", ["docs:read", "docs:clone"], { "/api/projects": [project] });

    await user.click(await screen.findByRole("button", { name: "More actions for Storage Spine" }));
    await user.click(await screen.findByRole("menuitem", { name: "Clone" }));
    const dialog = await screen.findByRole("dialog", { name: "Clone to…" });
    await user.click(await within(dialog).findByRole("button", { name: "Clone" }));

    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/clone", { project_id: "p-1" }));
    expect(await screen.findByText("editing doc-9")).toBeInTheDocument();
  });

  it("says so, and offers a writer New doc, when the project has no docs", async () => {
    renderPage("/acme/docs", ["docs:read", "docs:write"], { "/api/docs": [] });

    expect(await screen.findByText("No docs yet")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "New doc" })).toBeInTheDocument();
  });

  it("points at the project wizard when the workspace has no project yet", async () => {
    renderPage("/acme/docs", ["projects:write"], { "/api/projects": [] });

    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "New project" })).toHaveAttribute("href", "/acme/wizard/project/project");
  });

  it("shows the shared error display when the list fails", async () => {
    renderPage("/acme/docs", ["docs:read"], { "/api/docs": new Error("boom") });

    expect(await screen.findByText("Failed to load docs.")).toBeInTheDocument();
  });
});
