import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocPage } from "@/pages/DocPage";

vi.mock("@/components/doc/collab/useCollabSession", () => ({
  useCollabSession: () => null,
}));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
        <MemoryRouter initialEntries={["/docs/doc-1"]}>
          <Routes>
            <Route path="/docs/:docId" element={<DocPage />} />
          </Routes>
        </MemoryRouter>
    </QueryClientProvider>,
  );
};

const docData = {
  id: "doc-1",
  project_id: "p-1",
  title: "Storage Spine",
  body: "SQLite is the spine.",
  version: 2,
  archived: false,
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const projectData = { id: "p-1", name: "Backend", position: 0, created_at: "", updated_at: "" };

const ticketData = (id: string, title: string, docId: string) => ({
  id,
  project_id: "p-1",
  title,
  body: "",
  status: "open",
  doc_id: docId,
  assignee: "",
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
});

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DocPage", () => {
  it("renders the doc and its derived tickets", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/docs/doc-1") return Promise.resolve({ data: docData });
      if (url === "/api/tickets") return Promise.resolve({ data: [ticketData("t-1", "Write migrations", "doc-1")] });
      if (url.startsWith("/api/projects")) return Promise.resolve({ data: [projectData] });
      if (url.startsWith("/api/ticket-types"))
        return Promise.resolve({ data: [{ id: "ticket-type-task", name: "task", position: 0, created_at: "", updated_at: "" }] });
      if (url === "/api/pairing/presence") return Promise.resolve({ data: { computers: {} } });
      return Promise.resolve({ data: [] });
    });

    renderPage();
    expect(await screen.findByRole("heading", { name: "Storage Spine" })).toBeInTheDocument();
    // Actions and linked tickets live behind the settings-cog toggle.
    await user.click(await screen.findByRole("button", { name: "Doc actions" }));
    expect(await screen.findByText("Write migrations")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create ticket from this doc" })).toBeInTheDocument();
  });

  it("creates a ticket from the doc with the doc id pre-set", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/docs/doc-1") return Promise.resolve({ data: docData });
      if (url.startsWith("/api/projects")) return Promise.resolve({ data: [projectData] });
      if (url.startsWith("/api/ticket-types"))
        return Promise.resolve({ data: [{ id: "ticket-type-task", name: "task", position: 0, created_at: "", updated_at: "" }] });
      if (url === "/api/pairing/presence") return Promise.resolve({ data: { computers: {} } });
      return Promise.resolve({ data: [] });
    });
    vi.mocked(api.post).mockResolvedValue({ data: ticketData("t-1", "New task", "doc-1") });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Doc actions" }));
    await user.click(await screen.findByRole("button", { name: "Create ticket from this doc" }));
    await user.type(screen.getByLabelText("Title"), "New task");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(api.post).toHaveBeenCalledWith("/api/tickets", {
      title: "New task",
      body: "",
      project_id: "p-1",
      doc_id: "doc-1",
      assignee: "",
      category_id: "",
      type_id: "ticket-type-task",
    });
  });

  it("shows the shared error display", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("boom")).toBeInTheDocument();
  });
});
