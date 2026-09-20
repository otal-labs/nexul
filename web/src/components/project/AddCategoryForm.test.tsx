import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddCategoryForm } from "@/components/project/AddCategoryForm";
import { api } from "@/api/client";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveCategoryFormSchema, type SaveCategoryFormData } from "@/models/Category";
import { pickOption } from "@/test/pickOption";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

// Both call sites (Board's toolbar shortcut with no projectId, Projects' fixed-project panel) share this form (ADR 0006); the harness covers both shapes.
const CategoryHarness = ({ projectId }: { projectId?: string }) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<SaveCategoryFormData>({
            title: "New category",
            schema: SaveCategoryFormSchema,
            okLabel: "Create category",
            form: <AddCategoryForm {...(projectId !== undefined ? { projectId } : {})} />,
            formOptions: { defaultValues: { project_id: projectId ?? "", name: "", color: "" } },
          });
          setResult(result.success ? "created" : "cancelled");
        }}
      >
        Open
      </button>
      <span>{result}</span>
    </div>
  );
};

const renderHarness = (projectId?: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: projects };
    return { data: [] };
  });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <CategoryHarness {...(projectId !== undefined ? { projectId } : {})} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("AddCategoryForm", () => {
  it("shows a project picker and creates a category when no projectId is fixed", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "c-1", project_id: "p-2", name: "Sprint 1", position: 0, created_at: "", updated_at: "" },
    });
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await pickOption(user, "Project", "Frontend");
    await user.type(screen.getByLabelText("Name"), "Sprint 1");
    await user.click(screen.getByRole("button", { name: "Create category" }));

    expect(api.post).toHaveBeenCalledWith("/api/categories", { project_id: "p-2", name: "Sprint 1", color: "" });
    expect(await screen.findByText("created")).toBeInTheDocument();
  });

  it("hides the project picker when projectId is fixed", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "c-2", project_id: "p-1", name: "Bugs", position: 0, created_at: "", updated_at: "" },
    });
    renderHarness("p-1");

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.queryByLabelText("Project")).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Name"), "Bugs");
    await user.click(screen.getByRole("button", { name: "Create category" }));

    expect(api.post).toHaveBeenCalledWith("/api/categories", { project_id: "p-1", name: "Bugs", color: "" });
  });

  it("does not submit an empty category name", async () => {
    const user = userEvent.setup();
    renderHarness("p-1");

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(screen.getByRole("button", { name: "Create category" }));

    expect(await screen.findByText("Category name is required")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    // Validation failure leaves the dialog open; close it so Radix's pointer-events body lock doesn't leak into the next test.
    await user.keyboard("{Escape}");
  });
});
