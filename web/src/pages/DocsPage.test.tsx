import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocsPage } from "@/pages/DocsPage";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// The real wrapper lazy-imports CreateDocForm (tiptap + yjs); under full-suite load that dynamic import can outlive findBy's timeout, so the test resolves it synchronously.
vi.mock("@/components/doc/LazyCreateDocForm", async () => ({
  LazyCreateDocForm: (await import("@/components/doc/CreateDocForm")).CreateDocForm,
}));

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
        <MemoryRouter>
          <DocsPage />
        </MemoryRouter>
    </QueryClientProvider>,
  );
};

const listItem = {
  id: "doc-1",
  project_id: "p-1",
  title: "Storage Spine",
  version: 1,
  archived: false,
  can_open: true,
  updated_at: "2026-08-02T12:00:00Z",
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DocsPage", () => {
  it("renders docs from the gateway", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [listItem] });
    renderPage();
    expect(await screen.findByText("Storage Spine")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New doc" })).toBeInTheDocument();
  });

  it("shows the shared empty state when there are no docs", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    renderPage();
    expect(await screen.findByText("No docs yet.")).toBeInTheDocument();
    // the create action stays available on an empty page (first doc UX)
    expect(screen.getByRole("button", { name: "New doc" })).toBeInTheDocument();
  });

  it("creates a doc through the dialog", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url.startsWith("/api/projects")) return { data: [{ id: "p-1", name: "Backend", position: 0, created_at: "", updated_at: "" }] };
      return { data: [listItem] };
    });
    vi.mocked(api.post).mockResolvedValue({
      data: {
        id: "doc-1",
        project_id: "p-1",
        title: "New spec",
        body: "",
        version: 1,
        archived: false,
        created_at: "2026-08-02T12:00:00Z",
        updated_at: "2026-08-02T12:00:00Z",
      },
    });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "New doc" }));
    await user.type(await screen.findByLabelText("Title"), "New spec");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(api.post).toHaveBeenCalledTimes(1);
    const [url, payload] = vi.mocked(api.post).mock.calls[0] as [string, { project_id: string; title: string; body: string }];
    expect(url).toBe("/api/docs");
    expect(payload.project_id).toBe("p-1");
    expect(payload.title).toBe("New spec");
    expect(JSON.parse(payload.body)).toHaveProperty("type", "doc");
  });

  it("opens the permissions dialog for selected docs", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockResolvedValue({ data: [listItem, { ...listItem, id: "doc-2", title: "Event Bus" }] });
    renderPage();

    await user.click(await screen.findByLabelText("Select Storage Spine"));
    await user.click(screen.getByRole("button", { name: "Permissions (1)" }));
    expect(await screen.findByText("Permissions")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("shows the shared error display when the list fails", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("boom")).toBeInTheDocument();
  });
});
