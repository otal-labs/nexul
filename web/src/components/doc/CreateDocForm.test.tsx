import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// The real wrapper lazy-imports CreateDocForm (tiptap + yjs); resolving it synchronously keeps findBy inside its timeout.
vi.mock("@/components/doc/LazyCreateDocForm", async () => ({
  LazyCreateDocForm: (await import("@/components/doc/CreateDocForm")).CreateDocForm,
}));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const folders: Record<string, unknown[]> = {
  "p-1": [
    { id: "f-main", project_id: "p-1", name: "Main", is_default: true },
    { id: "f-gs", project_id: "p-1", name: "Get Source", is_default: false },
  ],
  "p-2": [{ id: "f-main-2", project_id: "p-2", name: "Main", is_default: true }],
};

const mockApi = (list: unknown[] = projects) => {
  vi.mocked(api.get).mockImplementation(async (url, config) => {
    if (url === "/api/projects") return { data: list };
    if (url === "/api/docs/folders") return { data: folders[(config?.params as { project_id: string } | undefined)?.project_id ?? ""] ?? [] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["docs:write"] } };
    return { data: [] };
  });
  vi.mocked(api.post).mockImplementation(async (url: string) => {
    if (url === "/api/docs") return { data: { id: "doc-1" } };
    return { data: { chips: [] } };
  });
};

const Opener = ({ folderId, onCreated }: { folderId?: string | undefined; onCreated?: ((id: string | undefined) => void) | undefined }) => {
  const open = useCreateDocDialog("p-1", folderId);
  return (
    <button type="button" onClick={() => void Promise.resolve(open?.()).then(onCreated)}>
      Open
    </button>
  );
};

const openDialog = async (folderId?: string, onCreated?: (id: string | undefined) => void) => {
  const user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <Opener folderId={folderId} onCreated={onCreated} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await user.click(await screen.findByRole("button", { name: "Open" }));
  return { user, dialog: await screen.findByRole("dialog", { name: "New doc" }) };
};

const docPosts = () => vi.mocked(api.post).mock.calls.filter(([url]) => url === "/api/docs");

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.put).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("New doc dialog", () => {
  it("creates a doc in the project picked in the header, with its title and body, on Ctrl+Enter", async () => {
    mockApi();
    const { user, dialog } = await openDialog();

    const title = await within(dialog).findByLabelText("Title");
    expect(title).toHaveFocus();
    await user.click(within(dialog).getByRole("button", { name: "Backend" }));
    await user.click(await screen.findByRole("button", { name: "Frontend" }));
    await user.type(title, "Rollback plan");
    const body = within(dialog).getByLabelText("Body");
    await user.click(body);
    // ProseMirror reads jsdom's DOM edits one key at a time; a burst of keys drops some.
    for (const key of "Drain first") await user.keyboard(key === " " ? "[Space]" : key);
    await user.keyboard("{Control>}{Enter}{/Control}");

    await vi.waitFor(() => expect(docPosts()).toHaveLength(1));
    const [, payload] = docPosts()[0] as [string, { project_id: string; title: string; body: string }];
    expect(payload.project_id).toBe("p-2");
    expect(payload.title).toBe("Rollback plan");
    expect(JSON.parse(payload.body)).toMatchObject({ type: "doc" });
    expect(payload.body).toContain("Drain first");
  });

  it("resolves with the id of the doc it created, and with nothing when cancelled", async () => {
    mockApi();
    const onCreated = vi.fn();
    const { user, dialog } = await openDialog(undefined, onCreated);
    await user.type(await within(dialog).findByLabelText("Title"), "Runbook");
    await user.click(within(dialog).getByRole("button", { name: "Create doc" }));
    await vi.waitFor(() => expect(onCreated).toHaveBeenCalledWith("doc-1"));

    await user.click(await screen.findByRole("button", { name: "Open" }));
    await within(await screen.findByRole("dialog", { name: "New doc" })).findByLabelText("Title");
    await user.keyboard("{Escape}");
    await vi.waitFor(() => expect(onCreated).toHaveBeenLastCalledWith(undefined));
  });

  it("files a doc started from a folder in that folder", async () => {
    mockApi();
    const { user, dialog } = await openDialog("f-gs");
    await user.type(await within(dialog).findByLabelText("Title"), "EP01");
    await user.click(within(dialog).getByRole("button", { name: "Create doc" }));
    await vi.waitFor(() => expect(docPosts()).toHaveLength(1));
    expect(docPosts()[0]?.[1]).toMatchObject({ project_id: "p-1", folder_id: "f-gs" });
  });

  it("files the doc in the folder picked in the header", async () => {
    mockApi();
    const { user, dialog } = await openDialog();
    await user.type(await within(dialog).findByLabelText("Title"), "EP02");
    await user.click(await within(dialog).findByRole("button", { name: "Main" }));
    await user.click(await screen.findByRole("button", { name: "Get Source" }));
    expect(within(dialog).getByRole("button", { name: "Get Source" })).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Create doc" }));
    await vi.waitFor(() => expect(docPosts()).toHaveLength(1));
    expect(docPosts()[0]?.[1]).toMatchObject({ project_id: "p-1", folder_id: "f-gs" });
  });

  it("drops the folder when another project is picked, which files the doc in that project's default folder", async () => {
    mockApi();
    const { user, dialog } = await openDialog("f-gs");
    await user.type(await within(dialog).findByLabelText("Title"), "Elsewhere");
    await user.click(within(dialog).getByRole("button", { name: "Backend" }));
    await user.click(await screen.findByRole("button", { name: "Frontend" }));
    await user.click(within(dialog).getByRole("button", { name: "Create doc" }));
    await vi.waitFor(() => expect(docPosts()).toHaveLength(1));
    expect(docPosts()[0]?.[1]).toMatchObject({ project_id: "p-2" });
    expect(docPosts()[0]?.[1]).not.toHaveProperty("folder_id");
  });

  it("rejects an empty title without submitting", async () => {
    mockApi();
    const { user, dialog } = await openDialog();

    await within(dialog).findByLabelText("Title");
    await user.click(within(dialog).getByRole("button", { name: "Create doc" }));

    expect(await within(dialog).findByText("Title is required")).toBeInTheDocument();
    expect(docPosts()).toHaveLength(0);
    await user.keyboard("{Escape}");
  });

  it("asks for a project when none exist", async () => {
    mockApi([]);
    const { user, dialog } = await openDialog();

    expect(await within(dialog).findByText(/create a project first/i)).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Create doc" })).toBeDisabled();
    await user.keyboard("{Escape}");
  });

  it("attaches an image pasted before the doc exists to the new doc and saves its path in the body", async () => {
    mockApi();
    globalThis.URL.createObjectURL = vi.fn(() => "blob:staged-1");
    globalThis.URL.revokeObjectURL = vi.fn();
    vi.mocked(api.post).mockImplementation(async (url: string) => {
      if (url === "/api/docs") return { data: { id: "doc-1", title: "Runbook" } };
      if (url === "/api/attachments") return { data: { id: "a-7", name: "shot.png", content_type: "image/png", size: 3 } };
      return { data: { chips: [] } };
    });
    vi.mocked(api.put).mockResolvedValue({ data: { id: "doc-1" } });
    const { user, dialog } = await openDialog();

    await user.type(await within(dialog).findByLabelText("Title"), "Runbook");
    fireEvent.paste(within(dialog).getByLabelText("Body"), {
      clipboardData: { files: [new File(["png"], "shot.png", { type: "image/png" })], items: [], types: ["Files"], getData: () => "" },
    });
    await user.click(within(dialog).getByRole("button", { name: "Create doc" }));

    await vi.waitFor(() => expect(api.put).toHaveBeenCalled());
    const upload = vi.mocked(api.post).mock.calls.find(([url]) => url === "/api/attachments")?.[1] as FormData;
    expect(upload.get("doc_id")).toBe("doc-1");
    expect(docPosts()[0]?.[1]).not.toMatchObject({ body: expect.stringContaining("blob:") });
    const [url, saved] = vi.mocked(api.put).mock.calls[0] as [string, { body: string }];
    expect(url).toBe("/api/docs/doc-1");
    expect(saved.body).toContain('"src":"/api/attachments/a-7"');
  });
});
