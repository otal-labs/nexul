import type { ReactElement } from "react";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { toast } from "sonner";

import { api } from "@/api/client";
import { CreateTicketFooter } from "@/components/ticket/CreateTicketFooter";
import { CreateTicketForm } from "@/components/ticket/CreateTicketForm";
import { emptyTicketForm } from "@/models/Ticket";
import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveTicketFormSchema, type SaveTicketFormData } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const projects = [
  { id: "p-1", name: "Backend", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", position: 1, created_at: "", updated_at: "" },
];

const categories = [
  { id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, created_at: "", updated_at: "" },
  { id: "c-2", project_id: "p-2", name: "Bugs", position: 0, created_at: "", updated_at: "" },
];

const ticketTypes = [
  { id: "ticket-type-task", name: "task", position: 0, created_at: "", updated_at: "" },
  { id: "ticket-type-bug", name: "bug", position: 1, created_at: "", updated_at: "" },
  { id: "ticket-type-feature", name: "feature", position: 2, created_at: "", updated_at: "" },
];

const templatedTypes = [
  { id: "tt-task", name: "task", position: 0, body_template: "## What needs doing\n\n", created_at: "", updated_at: "" },
  { id: "tt-bug", name: "bug", position: 1, body_template: "## Steps to reproduce\n\n", created_at: "", updated_at: "" },
  { id: "tt-feature", name: "feature", position: 2, body_template: "## Why\n\n", created_at: "", updated_at: "" },
];

const members = [
  { user_id: "u-alice", login: "alice", display_name: "", avatar_url: "" },
  { user_id: "u-bob", login: "bob", display_name: "", avatar_url: "" },
];

const mockReferenceData = (
  options: { projects?: unknown[]; categories?: unknown[]; ticketTypes?: unknown[]; docTitle?: string } = {},
) => {
  const {
    projects: mockProjects = projects,
    categories: mockCategories = categories,
    ticketTypes: mockTicketTypes = ticketTypes,
  } = options;
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url.endsWith("/people")) return { data: { people: members } };
    if (url.startsWith("/api/projects")) return { data: mockProjects };
    if (url.startsWith("/api/categories")) return { data: mockCategories };
    if (url.startsWith("/api/ticket-types")) return { data: mockTicketTypes };
    if (url.startsWith("/api/docs/")) return { data: { id: "doc-9", title: options.docTitle ?? "Runbook", project_id: "p-1" } };
    return { data: [] };
  });
};

interface TicketHarnessProps {
  docId?: string;
  defaultProjectId?: string;
  defaultCategoryId?: string;
}

// Mirrors how DocPage/BoardPage actually open the dialog: header (project chip) and footerStart
// (Create more) both live outside the `form` prop but share its react-hook-form instance.
const TicketHarness = ({ docId = "", defaultProjectId = "", defaultCategoryId = "" }: TicketHarnessProps) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<SaveTicketFormData>({
            title: "New ticket",
            schema: SaveTicketFormSchema,
            okLabel: "Create",
            header: <ProjectDialogHeader title="New ticket" />,
            footerStart: <CreateTicketFooter />,
            form: <CreateTicketForm docId={docId} defaultProjectId={defaultProjectId} defaultCategoryId={defaultCategoryId} />,
            formOptions: { defaultValues: emptyTicketForm() },
          });
          setResult(result.success ? String((result.data as SaveTicketFormData & { id?: string })?.id) : "cancelled");
        }}
      >
        Open
      </button>
      <p>{result}</p>
    </div>
  );
};

const renderWithRoot = (ui: ReactElement) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      {ui}
    </QueryClientProvider>,
  );

// ProseMirror reads jsdom's DOM edits one key at a time; a burst of keys drops some.
const typeInBody = async (user: ReturnType<typeof userEvent.setup>, text: string) => {
  await user.click(screen.getByLabelText("Body"));
  for (const key of text) await user.keyboard(key === " " ? "[Space]" : key);
};

const pasteFiles = (files: File[]) =>
  fireEvent.paste(screen.getByLabelText("Body"), {
    clipboardData: { files, items: [], types: ["Files"], getData: () => "" },
  });

// The created ticket is t-20; each upload is answered by name, and shot-fail.png is refused.
const mockCreateWithUploads = () => {
  let objectUrls = 0;
  globalThis.URL.createObjectURL = vi.fn(() => `blob:staged-${++objectUrls}`);
  globalThis.URL.revokeObjectURL = vi.fn();
  vi.mocked(api.post).mockImplementation(async (url: string, payload: unknown) => {
    if (url === "/api/tickets") return { data: { id: "t-20", title: "With a screenshot" } };
    if (url !== "/api/attachments") return { data: { chips: [] } };
    const name = (payload as FormData).get("name");
    if (name === "shot-fail.png") throw new Error("too large");
    return { data: { id: `a-${String(name)}`, name, content_type: "image/png", size: 3 } };
  });
  vi.mocked(api.patch).mockResolvedValue({ data: { id: "t-20" } });
};

const uploadedOwners = () =>
  vi
    .mocked(api.post)
    .mock.calls.filter(([url]) => url === "/api/attachments")
    .map(([, form]) => (form as FormData).get("ticket_id"));

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(toast.error).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useWorkspaceStore.persist.clearStorage();
});

describe("CreateTicketForm", () => {
  it("does not show the empty state while projects are still loading", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}));
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.queryByText(/Create a project first/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    await user.keyboard("{Escape}");
  });

  it("creates a ticket in the selected project and resolves with its id", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-9", title: "Fix storage", project_id: "p-1", status: "open" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "Fix storage");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("t-9")).toBeInTheDocument();
    expect(api.post).toHaveBeenCalledWith("/api/tickets", {
      title: "Fix storage",
      body: "",
      project_id: "p-1",
      doc_id: "",
      developer: "",
      tester: "",
      category_id: "",
      type_id: "ticket-type-task",
    });
  });

  it("presets the doc id and a chosen default project when deriving from a doc", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-10" } });
    renderWithRoot(<TicketHarness docId="doc-9" defaultProjectId="p-2" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "Fix from doc");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/tickets", {
        title: "Fix from doc",
        body: "",
        project_id: "p-2",
        doc_id: "doc-9",
        developer: "",
        tester: "",
        category_id: "",
        type_id: "ticket-type-task",
      }),
    );
  });

  it("lets the user pick a category scoped to the project", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-11" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "In Sprint");
    await user.click(await screen.findByRole("button", { name: "Category" }));
    await user.click(await screen.findByRole("button", { name: "Sprint 1" }));
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/tickets", expect.objectContaining({ title: "In Sprint", category_id: "c-1" })),
    );
  });

  it("lets the user pick a ticket type", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-12" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "A feature");
    await user.click(await screen.findByRole("button", { name: "task" }));
    expect(screen.queryByRole("button", { name: "bug" })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "feature" }));
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/tickets", expect.objectContaining({ type_id: "ticket-type-feature" })),
    );
  });

  it("lets the user search and pick a developer and a tester, storing their logins", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-13" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "Needs an owner");
    await user.click(await screen.findByRole("button", { name: "Developer: no one" }));
    await user.type(await screen.findByLabelText("Search people"), "bo");
    expect(screen.queryByRole("button", { name: /alice/ })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: /bob/ }));
    await user.click(await screen.findByRole("button", { name: "Tester: no one" }));
    await user.click(await screen.findByRole("button", { name: /alice/ }));
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        "/api/tickets",
        expect.objectContaining({ developer: "bob", tester: "alice" }),
      ),
    );
  });

  it("shows a removable doc chip and clears doc_id when it's removed", async () => {
    const user = userEvent.setup();
    mockReferenceData({ docTitle: "Runbook" });
    renderWithRoot(<TicketHarness docId="doc-9" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByText("Runbook")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove doc link" }));
    expect(screen.queryByText("Runbook")).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("re-seeds the type and clears a category that belonged to the old project on a project switch", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    renderWithRoot(<TicketHarness defaultCategoryId="c-1" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByRole("button", { name: "Sprint 1" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Backend" }));
    await user.click(await screen.findByRole("button", { name: "Frontend" }));

    expect(await screen.findByRole("button", { name: "Category" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sprint 1" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("rejects an empty title", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await screen.findByLabelText("Title");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(screen.getByText("Title is required")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.keyboard("{Escape}");
  });

  it("asks for a project when none exist", async () => {
    const user = userEvent.setup();
    mockReferenceData({ projects: [] });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByText(/create a project first/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    await user.keyboard("{Escape}");
  });

  it("submits on Cmd/Ctrl+Enter", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-14" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "Keyboard submit");
    await user.keyboard("{Control>}{Enter}{/Control}");

    expect(await screen.findByText("t-14")).toBeInTheDocument();
  });

  it("pre-fills the body from the type and swaps it on a type change while untouched", async () => {
    const user = userEvent.setup();
    mockReferenceData({ ticketTypes: templatedTypes });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await vi.waitFor(() => expect(screen.getByLabelText("Body")).toHaveTextContent("What needs doing"));
    await user.click(await screen.findByRole("button", { name: "task" }));
    await user.click(await screen.findByRole("button", { name: "feature" }));

    await vi.waitFor(() => expect(screen.getByLabelText("Body")).toHaveTextContent("Why"));
    expect(screen.getByLabelText("Body")).not.toHaveTextContent("What needs doing");
    await user.keyboard("{Escape}");
  });

  it("keeps an edited body on a type change", async () => {
    const user = userEvent.setup();
    mockReferenceData({ ticketTypes: templatedTypes });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    const body = await screen.findByLabelText("Body");
    await vi.waitFor(() => expect(body).toHaveTextContent("What needs doing"));
    await typeInBody(user, "ship it");
    await user.click(await screen.findByRole("button", { name: "task" }));
    await user.click(await screen.findByRole("button", { name: "feature" }));

    expect(body).toHaveTextContent("What needs doing");
    expect(body).toHaveTextContent("ship it");
    expect(body).not.toHaveTextContent("Why");
    await user.keyboard("{Escape}");
  });

  it("resets the body to the type's template after a Create more submit", async () => {
    const user = userEvent.setup();
    mockReferenceData({ ticketTypes: templatedTypes });
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-16" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(await screen.findByRole("checkbox", { name: "Create more" }));
    const body = await screen.findByLabelText("Body");
    await vi.waitFor(() => expect(body).toHaveTextContent("What needs doing"));
    await user.type(await screen.findByLabelText("Title"), "First ticket");
    await typeInBody(user, "done");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
    await vi.waitFor(() => expect(screen.getByLabelText("Body")).not.toHaveTextContent("done"));
    expect(screen.getByLabelText("Body")).toHaveTextContent("What needs doing");
    await user.keyboard("{Escape}");
  });

  it("keeps the dialog open, clears title/body, and keeps everything else when Create more is checked", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-15" } });
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(await screen.findByRole("button", { name: "Category" }));
    await user.click(await screen.findByRole("button", { name: "Sprint 1" }));
    await user.click(screen.getByRole("checkbox", { name: "Create more" }));

    const titleInput = await screen.findByLabelText("Title");
    await user.type(titleInput, "First ticket");
    await typeInBody(user, "First body");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(await screen.findByLabelText("Title")).toHaveValue("");
    await vi.waitFor(() => expect(screen.getByLabelText("Body")).not.toHaveTextContent("First body"));
    // Non-title/body fields survive the reset, so a batch of tickets can share the same category.
    expect(screen.getByRole("button", { name: "Sprint 1" })).toBeInTheDocument();

    await user.keyboard("{Escape}");
  });

  it("attaches an image pasted before the ticket exists to the new ticket and saves its path in the body", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    mockCreateWithUploads();
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "With a screenshot");
    pasteFiles([new File(["png"], "shot.png", { type: "image/png" })]);
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("t-20")).toBeInTheDocument();
    expect(uploadedOwners()).toEqual(["t-20"]);
    const saved = vi.mocked(api.patch).mock.calls.find(([url]) => url === "/api/tickets/t-20")?.[1] as { body: string };
    expect(saved.body).toContain('"src":"/api/attachments/a-shot.png"');
    expect(saved.body).not.toContain("blob:");
  });

  it("keeps the ticket and the other images when one staged upload fails, naming the failed file", async () => {
    const user = userEvent.setup();
    mockReferenceData();
    mockCreateWithUploads();
    renderWithRoot(<TicketHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "With a screenshot");
    pasteFiles([new File(["png"], "shot-fail.png", { type: "image/png" }), new File(["png"], "shot.png", { type: "image/png" })]);
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("t-20")).toBeInTheDocument();
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("shot-fail.png"));
    const saved = vi.mocked(api.patch).mock.calls.find(([url]) => url === "/api/tickets/t-20")?.[1] as { body: string };
    expect(saved.body).toContain('"src":"/api/attachments/a-shot.png"');
    expect(saved.body).not.toContain("blob:");
  });
});
