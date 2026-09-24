import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectCategories } from "@/components/project/ProjectCategories";
import { api } from "@/api/client";
import type { Category } from "@/models/Category";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const categories: Category[] = [
  { id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, color: "", created_at: "", updated_at: "" },
  { id: "c-2", project_id: "p-1", name: "Bugs", position: 1, color: "", created_at: "", updated_at: "" },
];

const tickets = [
  {
    id: "t-1",
    project_id: "p-1",
    category_id: "c-1",
    type_id: "ticket-type-task",
    title: "Fix auth",
    body: "",
    status: "open",
    doc_id: "",
    developer: "onik97",
    tester: "",
    reporter: { kind: "user", login: "onik97" },
    created_at: "",
    updated_at: "",
    labels: [],
  },
  {
    id: "t-2",
    project_id: "p-1",
    category_id: "c-2",
    type_id: "ticket-type-task",
    title: "Add tests",
    body: "",
    status: "open",
    doc_id: "",
    developer: "alice",
    tester: "",
    reporter: { kind: "user", login: "onik97" },
    created_at: "",
    updated_at: "",
    labels: [],
  },
  {
    id: "t-3",
    project_id: "p-1",
    category_id: "c-2",
    type_id: "ticket-type-task",
    title: "Docs",
    body: "",
    status: "open",
    doc_id: "",
    developer: "",
    tester: "",
    reporter: { kind: "user", login: "onik97" },
    created_at: "",
    updated_at: "",
    labels: [],
  },
];

const renderCategories = (list: Category[] = categories) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/categories") return { data: list };
    if (url === "/api/tickets") return { data: tickets };
    return { data: [] };
  });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ProjectCategories projectId="p-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("ProjectCategories", () => {
  it("lists categories with ticket count chips", async () => {
    renderCategories();

    expect(await screen.findByText("Sprint 1")).toBeInTheDocument();
    expect(screen.getByText("Bugs")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("creates a category through the dialog", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "c-3", project_id: "p-1", name: "Ops", position: 2, created_at: "", updated_at: "" } });
    renderCategories();

    await user.click(await screen.findByRole("button", { name: "New category" }));
    await user.type(screen.getByLabelText("Name"), "Ops");
    await user.click(screen.getByRole("button", { name: "Create category" }));
    expect(api.post).toHaveBeenCalledWith("/api/categories", { project_id: "p-1", name: "Ops", color: "" });
  });

  it("shows an empty state when there are no categories", async () => {
    renderCategories([]);
    expect(
      await screen.findByText("No categories yet — the board groups tickets into swimlanes per category."),
    ).toBeInTheDocument();
  });

  it("does not show the empty state while categories are still loading", () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}));
    renderCategories([]);
    expect(
      screen.queryByText("No categories yet — the board groups tickets into swimlanes per category."),
    ).not.toBeInTheDocument();
  });

  it("renames a category inline via the kebab menu's Edit action", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: { ...categories[0]!, name: "Sprint 2" } });
    renderCategories();

    await user.click(await screen.findByRole("button", { name: "Actions for Sprint 1" }));
    await user.click(await screen.findByRole("button", { name: "Edit" }));
    const input = screen.getByLabelText("Category name");
    await user.clear(input);
    await user.type(input, "Sprint 2");
    await user.keyboard("{Enter}");
    expect(api.patch).toHaveBeenCalledWith("/api/categories/c-1", { name: "Sprint 2", color: "" });
  });

  it("reorders categories from the kebab menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    renderCategories();

    await user.click(await screen.findByRole("button", { name: "Actions for Sprint 1" }));
    await user.click(await screen.findByRole("button", { name: "Move down" }));
    expect(api.post).toHaveBeenCalledWith("/api/categories/reorder", { project_id: "p-1", ids: ["c-2", "c-1"] });
  });

  it("disables Move up for the first row and Move down for the last row in the kebab menu", async () => {
    const user = userEvent.setup();
    renderCategories();

    await user.click(await screen.findByRole("button", { name: "Actions for Sprint 1" }));
    expect(screen.getByRole("button", { name: "Move up" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Move down" })).not.toBeDisabled();
    await user.keyboard("{Escape}");

    await user.click(screen.getByRole("button", { name: "Actions for Bugs" }));
    expect(screen.getByRole("button", { name: "Move down" })).toBeDisabled();
  });

  it("removes a category from the kebab menu", async () => {
    const user = userEvent.setup();
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    renderCategories();

    await user.click(await screen.findByRole("button", { name: "Actions for Sprint 1" }));
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    expect(api.delete).toHaveBeenCalledWith("/api/categories/c-1");
  });
});
