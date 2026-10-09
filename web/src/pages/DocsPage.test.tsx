import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocsPage } from "@/pages/DocsPage";
import { usePinnedDocStore } from "@/stores/pinnedDocStore";
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

const main = { id: "f-main", project_id: "p-1", name: "Main", is_default: true, created_at: "", updated_at: "" };
const getSource = { id: "f-gs", project_id: "p-1", name: "GetSource", is_default: false, created_at: "", updated_at: "" };

const doc = (id: string, title: string, updated_at: string, can_open = true, locked = false, folder_id = "f-main") => ({
  id,
  project_id: "p-1",
  folder_id,
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
  doc("doc-2", "Rollback plan", "2020-01-01T12:00:00Z", true, false, "f-gs"),
  doc("doc-3", "Salaries", "2020-01-02T12:00:00Z", false),
];

const LocationSpy = () => <p data-testid="location">{useLocation().pathname}</p>;

const renderPage = (path: string, permissions: string[], overrides: Record<string, unknown> = {}) => {
  const endpoints: Record<string, unknown> = {
    "/api/projects": [project],
    "/api/workspaces": [{ id: "ws-1", name: "Acme", slug: "acme" }],
    "/api/workspaces/ws-1/me": { role_name: "Member", permissions },
    "/api/docs": docs,
    "/api/docs/folders": [main, getSource],
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
  usePinnedDocStore.setState({ pinned: {} });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DocsPage", () => {
  it("lists the openable docs by folder and opens the one the URL names", async () => {
    renderPage("/acme/docs/BE/doc-2", ["docs:read"]);

    const folder = await screen.findByRole("region", { name: "GetSource" });
    expect(within(folder).getByRole("link", { name: /Rollback plan/ })).toHaveAttribute("aria-current", "page");
    const mainFolder = screen.getByRole("region", { name: "Main" });
    expect(within(mainFolder).getByRole("link", { name: /Storage Spine/ })).not.toHaveAttribute("aria-current");
    expect(screen.queryByText("Salaries")).not.toBeInTheDocument();
    expect(await screen.findByText("editing doc-2")).toBeInTheDocument();
  });

  it("says the project has no docs instead of asking to pick one", async () => {
    renderPage("/acme/docs", ["docs:read"], { "/api/docs": [] });

    expect(await screen.findByText("Nothing to open yet")).toBeInTheDocument();
    expect(screen.queryByText("Select a doc")).not.toBeInTheDocument();
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

  it("leaves Lock out for a writer without docs:lock", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Rollback plan" }));
    expect(await screen.findByRole("menuitem", { name: "Move to folder" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Lock" })).not.toBeInTheDocument();
  });

  it("locks and unlocks a doc from its row menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    renderPage("/acme/docs", ["docs:read", "docs:write", "docs:clone", "docs:delete", "docs:lock"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Rollback plan" }));
    const items = await screen.findAllByRole("menuitem");
    expect(items.map((item) => item.textContent)).toEqual(["Pin", "Lock", "Move to folder", "Clone", "Delete"]);
    await user.click(screen.getByRole("menuitem", { name: "Lock" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-2/lock"));

    await user.click(screen.getByRole("button", { name: "More actions for Storage Spine" }));
    await user.click(await screen.findByRole("menuitem", { name: "Unlock" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/unlock"));
  });

  it("moves a doc into Pinned from its row menu and back out on Unpin", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Rollback plan" }));
    await user.click(await screen.findByRole("menuitem", { name: "Pin" }));
    const pinned = await screen.findByRole("region", { name: "Pinned" });
    expect(within(pinned).getByRole("link", { name: /Rollback plan/ })).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "GetSource" })).queryByRole("link")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "More actions for Rollback plan" }));
    await user.click(await screen.findByRole("menuitem", { name: "Unpin" }));
    expect(await within(screen.getByRole("region", { name: "GetSource" })).findByRole("link", { name: /Rollback plan/ })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Pinned" })).not.toBeInTheDocument();
  });

  it("clones a doc into the project picked in Clone to…", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "doc-9", project_id: "p-1" } });
    renderPage("/acme/docs", ["docs:read", "docs:clone"], { "/api/projects": [project] });

    await user.click(await screen.findByRole("button", { name: "More actions for Storage Spine" }));
    await user.click(await screen.findByRole("menuitem", { name: "Clone" }));
    const dialog = await screen.findByRole("dialog", { name: "Clone to…" });
    await user.click(await within(dialog).findByRole("button", { name: "Clone doc" }));

    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/clone", { project_id: "p-1" }));
    expect(await screen.findByText("editing doc-9")).toBeInTheDocument();
  });

  it("shows a writer the empty folders, with New doc, when the project has no docs", async () => {
    renderPage("/acme/docs", ["docs:read", "docs:write"], { "/api/docs": [] });

    expect(await within(await screen.findByRole("region", { name: "Main" })).findByText("No docs")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "New doc" })).toBeInTheDocument();
  });

  it("says so when the viewer can open no doc, so the server shows them no folder", async () => {
    renderPage("/acme/docs", ["docs:read"], { "/api/docs": [], "/api/docs/folders": [] });

    expect(await screen.findByText("Nothing to open yet")).toBeInTheDocument();
  });

  it("moves a doc to another folder from its row menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Storage Spine" }));
    await user.click(await screen.findByRole("menuitem", { name: "Move to folder" }));
    screen.getByRole("menuitemradio", { name: "GetSource" }).focus();
    await user.keyboard("{Enter}");
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/move", { folder_id: "f-gs" }));
  });

  it("creates a folder from the header and deletes one after saying where its docs go", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    vi.mocked(api.delete).mockResolvedValue({ data: {} });
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.click(await screen.findByRole("button", { name: "New folder" }));
    const dialog = await screen.findByRole("dialog", { name: "New folder" });
    await user.type(within(dialog).getByLabelText("Name"), "Episodes");
    await user.click(within(dialog).getByRole("button", { name: "Create folder" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/folders", { project_id: "p-1", name: "Episodes" }));

    await user.click(screen.getByRole("button", { name: "More actions for GetSource" }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
    const confirm = await screen.findByRole("dialog", { name: "Delete GetSource?" });
    expect(within(confirm).getByText("1 doc moves to Main. None are deleted.")).toBeInTheDocument();
    await user.click(within(confirm).getByRole("button", { name: "Delete folder" }));
    await vi.waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/docs/folders/f-gs"));
  });

  it("never offers to delete the default folder", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.click(await screen.findByRole("button", { name: "More actions for Main" }));
    expect(await screen.findByRole("menuitem", { name: "Rename" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Delete" })).not.toBeInTheDocument();
  });

  it("points at the project wizard when the workspace has no project yet", async () => {
    renderPage("/acme/docs", ["projects:write"], { "/api/projects": [] });

    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "New project" })).toHaveAttribute("href", "/acme/wizard/project/project");
  });

  it("shows the shared error display when the list fails", async () => {
    renderPage("/acme/docs", ["docs:read"], { "/api/docs": new Error("boom") });

    expect(await screen.findByText("Couldn't load docs.")).toBeInTheDocument();
  });
});
