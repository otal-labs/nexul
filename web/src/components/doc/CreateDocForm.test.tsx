import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
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

const mockApi = (list: unknown[] = projects) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: list };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["docs:write"] } };
    return { data: [] };
  });
  vi.mocked(api.post).mockImplementation(async (url: string) => {
    if (url === "/api/docs") return { data: { id: "doc-1" } };
    return { data: { chips: [] } };
  });
};

const Opener = () => {
  const open = useCreateDocDialog("p-1");
  return (
    <button type="button" onClick={open}>
      Open
    </button>
  );
};

const openDialog = async () => {
  const user = userEvent.setup();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <Opener />
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

  it("rejects an empty title without submitting", async () => {
    mockApi();
    const { user, dialog } = await openDialog();

    await within(dialog).findByLabelText("Title");
    await user.click(within(dialog).getByRole("button", { name: "Create" }));

    expect(await within(dialog).findByText("Title is required")).toBeInTheDocument();
    expect(docPosts()).toHaveLength(0);
    await user.keyboard("{Escape}");
  });

  it("asks for a project when none exist", async () => {
    mockApi([]);
    const { user, dialog } = await openDialog();

    expect(await within(dialog).findByText(/create a project first/i)).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Create" })).toBeDisabled();
    await user.keyboard("{Escape}");
  });
});
