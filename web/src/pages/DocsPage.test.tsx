import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocsPage } from "@/pages/DocsPage";
import { useDocArchiveStore } from "@/stores/docArchiveStore";
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

// The overview beside the list names the same docs and folders; a row's menu lives on the list's copy.
const inList = (text: string) =>
  vi.waitFor(() => {
    const own = screen.getAllByText(text).find((el) => !el.closest("[aria-label='Docs overview']"));
    if (!own) throw new Error(`${text} is not in the list yet`);
    return own;
  });

const LocationSpy = () => <p data-testid="location">{useLocation().pathname}</p>;

const renderPage = (path: string, permissions: string[], overrides: Record<string, unknown> = {}) => {
  const endpoints: Record<string, unknown> = {
    "/api/projects": [project],
    "/api/workspaces": [{ id: "ws-1", name: "Acme", slug: "acme" }],
    "/api/workspaces/ws-1/me": { role_name: "Member", permissions },
    "/api/docs": docs,
    "/api/docs/folders": [main, getSource],
    "/api/permissions/explain": { effective: [] },
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

const manageable = { "/api/permissions/explain": { effective: [{ action: "permissions:write", allowed: true }] } };

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  usePinnedDocStore.setState({ pinned: {} });
  useDocArchiveStore.setState({ showArchived: false });
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

  it.each([
    ["the project has no docs", { "/api/docs": [] }],
    ["the viewer can open no doc, so the server shows them no folder", { "/api/docs": [], "/api/docs/folders": [] }],
  ])("says the project has no docs yet, instead of asking to pick a doc, when %s", async (_, overrides) => {
    renderPage("/acme/docs", ["docs:read"], overrides);

    expect(await within(await screen.findByRole("region", { name: "Docs overview" })).findByText("No docs yet")).toBeInTheDocument();
    expect(screen.queryByText("Select a doc")).not.toBeInTheDocument();
  });

  it("narrows the overview to what the list's search matches", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read"]);
    const overview = await screen.findByRole("region", { name: "Docs overview" });
    await within(overview).findByText("Rollback plan");

    await user.type(screen.getByRole("textbox", { name: "Search docs" }), "storage");

    const matching = within(overview).getByRole("region", { name: "Matching" });
    expect(within(matching).getByText("Storage Spine")).toBeInTheDocument();
    expect(within(overview).queryByText("Rollback plan")).not.toBeInTheDocument();
  });

  it("shows the archived docs in the overview while the list shows them", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read"], { "/api/docs": [...docs, { ...doc("doc-4", "Old migration", "2020-03-01T12:00:00Z"), archived: true }] });
    const overview = await screen.findByRole("region", { name: "Docs overview" });
    await within(overview).findByText("Rollback plan");

    await user.click(screen.getByRole("button", { name: /^Sort and view/ }));
    await user.click(screen.getByRole("menuitemcheckbox", { name: "Show archived" }));

    const archived = within(overview).getByRole("region", { name: "Archived" });
    expect(within(archived).getByText("Old migration")).toBeInTheDocument();
    expect(within(overview).queryByText("Rollback plan")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Sort and view: Archived/ })).toBeInTheDocument();
  });

  it("moves a bare doc link to its project's URL", async () => {
    renderPage("/acme/docs/doc-2", ["docs:read"], { "/api/docs/doc-2": { ...doc("doc-2", "Rollback plan", ""), body: "" } });

    await vi.waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/acme/docs/BE/doc-2"));
    expect(await screen.findByText("editing doc-2")).toBeInTheDocument();
  });

  it("shows each row as its title and edited time, marking a locked one", async () => {
    renderPage("/acme/docs", ["docs:read"]);

    const row = (await within(await screen.findByRole("region", { name: "Main" })).findByRole("link", { name: /Storage Spine/ })).closest("li");
    expect(row).toHaveTextContent("Storage Spinenow");
    expect(within(row!).getByLabelText("Locked")).toBeInTheDocument();
    const other = within(screen.getByRole("region", { name: "GetSource" })).getByRole("link", { name: /Rollback plan/ }).closest("li");
    expect(other).toHaveTextContent(/^Rollback plan\S+ \S+$/);
    expect(within(other!).queryByLabelText("Locked")).not.toBeInTheDocument();
  });

  it("offers New doc and each row action only to a role that holds them", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:delete"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Storage Spine") });
    expect(await screen.findByRole("menuitem", { name: "Delete" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Unlock" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Clone" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New doc" })).not.toBeInTheDocument();
  });

  it("leaves Lock out for a writer without docs:lock", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Rollback plan") });
    expect(await screen.findByRole("menuitem", { name: "Move to folder" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Lock" })).not.toBeInTheDocument();
  });

  it("locks and unlocks a doc from its row menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    renderPage("/acme/docs", ["docs:read", "docs:write", "docs:clone", "docs:delete", "docs:lock"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Rollback plan") });
    const items = await screen.findAllByRole("menuitem");
    expect(items.map((item) => item.textContent)).toEqual(["Pin", "Lock", "Archive", "Move to folder", "Clone", "Delete"]);
    await user.click(screen.getByRole("menuitem", { name: "Lock" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-2/lock"));

    await user.pointer({ keys: "[MouseRight]", target: await inList("Storage Spine") });
    await user.click(await screen.findByRole("menuitem", { name: "Unlock" }));
    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/unlock"));
  });

  it("opens the row menu at the pointer on a right-click, with Permissions… for whoever may manage the doc's access", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:delete"], manageable);

    await user.pointer({ keys: "[MouseRight]", target: await within(await screen.findByRole("region", { name: "GetSource" })).findByRole("link", { name: /Rollback plan/ }) });
    expect(await screen.findByRole("menuitem", { name: "Permissions…" })).toBeInTheDocument();
    expect(screen.getAllByRole("menuitem").map((item) => item.textContent)).toEqual(["Pin", "Permissions…", "Delete"]);
  });

  it.each([
    ["a writer", ["docs:read", "docs:write"], ["New doc", "Rename", "Permissions…", "Delete"], false],
    ["a reader", ["docs:read"], ["Permissions…"], true],
  ])("offers %s who manages a folder's access Permissions… on its right-click menu", async (_, permissions, items, needsAccessRead) => {
    const user = userEvent.setup();
    renderPage("/acme/docs", permissions, manageable);

    // A reader's folder has no menu until the access read lands.
    if (needsAccessRead) await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/permissions/explain", expect.anything()));
    await user.pointer({ keys: "[MouseRight]", target: await inList("GetSource") });
    expect(await screen.findByRole("menuitem", { name: "Permissions…" })).toBeInTheDocument();
    expect(screen.getAllByRole("menuitem").map((item) => item.textContent)).toEqual(items);
    expect(api.get).toHaveBeenCalledWith("/api/permissions/explain", { params: { resource_type: "folder", resource_id: "f-gs" } });
  });

  it("keeps the browser's own menu on a folder for a reader who can't manage its access", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read"]);

    await vi.waitFor(() =>
      expect(api.get).toHaveBeenCalledWith("/api/permissions/explain", { params: { resource_type: "folder", resource_id: "f-gs" } }),
    );
    await user.pointer({ keys: "[MouseRight]", target: await inList("GetSource") });
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("leaves Permissions… out for someone who can't manage the doc's access", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:delete"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Rollback plan") });
    await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/permissions/explain", expect.anything()));
    expect(await screen.findByRole("menuitem", { name: "Delete" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Permissions…" })).not.toBeInTheDocument();
  });

  it("moves a doc into Pinned from its row menu and back out on Unpin", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Rollback plan") });
    await user.click(await screen.findByRole("menuitem", { name: "Pin" }));
    const pinned = await screen.findByRole("region", { name: "Pinned" });
    expect(within(pinned).getByRole("link", { name: /Rollback plan/ })).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "GetSource" })).queryByRole("link")).not.toBeInTheDocument();

    await user.pointer({ keys: "[MouseRight]", target: await inList("Rollback plan") });
    await user.click(await screen.findByRole("menuitem", { name: "Unpin" }));
    expect(await within(screen.getByRole("region", { name: "GetSource" })).findByRole("link", { name: /Rollback plan/ })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Pinned" })).not.toBeInTheDocument();
  });

  it("clones a doc into the project picked in Clone to…", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "doc-9", project_id: "p-1" } });
    renderPage("/acme/docs", ["docs:read", "docs:clone"], { "/api/projects": [project] });

    await user.pointer({ keys: "[MouseRight]", target: await inList("Storage Spine") });
    await user.click(await screen.findByRole("menuitem", { name: "Clone" }));
    const dialog = await screen.findByRole("dialog", { name: "Clone to…" });
    await user.click(await within(dialog).findByRole("button", { name: "Clone doc" }));

    await vi.waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/clone", { project_id: "p-1" }));
    expect(await screen.findByText("editing doc-9")).toBeInTheDocument();
  });

  it("offers a writer New doc in the overview and the list header when the project has no docs", async () => {
    renderPage("/acme/docs", ["docs:read", "docs:write"], { "/api/docs": [] });

    const overview = await screen.findByRole("region", { name: "Docs overview" });
    expect(await within(overview).findByRole("button", { name: "New doc" })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Main" })).not.toBeInTheDocument();
  });

  it("moves a doc to another folder from its row menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Storage Spine") });
    const moveTo = await screen.findByRole("menuitem", { name: "Move to folder" });
    await user.click(moveTo);
    const target = screen.getByRole("menuitemradio", { name: "GetSource" });
    // The row menu scrolls and blurs its backdrop, which would clip a submenu nested inside it.
    expect(moveTo.closest("[role=menu]")).not.toContainElement(target);
    target.focus();
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

    await user.pointer({ keys: "[MouseRight]", target: await inList("GetSource") });
    await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
    const confirm = await screen.findByRole("dialog", { name: "Delete GetSource?" });
    expect(within(confirm).getByText("1 doc moves to Main. None are deleted.")).toBeInTheDocument();
    await user.click(within(confirm).getByRole("button", { name: "Delete folder" }));
    await vi.waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/docs/folders/f-gs"));
  });

  it("never offers to delete the default folder", async () => {
    const user = userEvent.setup();
    renderPage("/acme/docs", ["docs:read", "docs:write"]);

    await user.pointer({ keys: "[MouseRight]", target: await inList("Main") });
    expect(await screen.findByRole("menuitem", { name: "Rename" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Delete" })).not.toBeInTheDocument();
  });

  it("points at the project wizard when the workspace has no project yet", async () => {
    renderPage("/acme/docs", ["projects:create", "projects:write", "stacks:write", "deploys:write"], { "/api/projects": [] });

    expect(await screen.findByText("No projects yet")).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "New project" })).toHaveAttribute("href", "/acme/wizard/project/project");
  });

  it("shows the shared error display when the list fails", async () => {
    renderPage("/acme/docs", ["docs:read"], { "/api/docs": new Error("boom") });

    expect(await screen.findByText("Couldn't load docs.")).toBeInTheDocument();
  });
});
