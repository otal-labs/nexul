import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CreateProjectForm } from "@/components/project/CreateProjectForm";
import { api } from "@/api/client";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveProjectFormSchema, type SaveProjectFormData } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ProjectHarness = () => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<SaveProjectFormData>({
            title: "New project",
            schema: SaveProjectFormSchema,
            okLabel: "Create project",
            form: <CreateProjectForm />,
            formOptions: { defaultValues: { name: "", prefix: "", icon: "" } },
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

const renderHarness = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ProjectHarness />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.post).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("CreateProjectForm", () => {
  it("creates a project on submit", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "p-3", name: "Docs", prefix: "DOC", position: 2, created_at: "", updated_at: "" },
    });
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Project name"), "Docs");
    await user.type(screen.getByLabelText("Prefix"), "DOC");
    await user.click(screen.getByRole("button", { name: "Create project" }));
    expect(api.post).toHaveBeenCalledWith("/api/projects", {
      name: "Docs",
      prefix: "DOC",
      icon: "",
      workspace_id: "ws-1",
    });
  });

  it("does not submit an empty name", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Prefix"), "DOC");
    await user.click(screen.getByRole("button", { name: "Create project" }));
    expect(await screen.findByText("Project name is required")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    // Validation failure leaves the dialog open; close it so Radix's pointer-events body lock doesn't leak into the next test.
    await user.keyboard("{Escape}");
  });

  it("rejects a malformed prefix", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Project name"), "Docs");
    await user.type(screen.getByLabelText("Prefix"), "D1");
    await user.click(screen.getByRole("button", { name: "Create project" }));
    expect(await screen.findByText("Prefix must be 2-5 letters")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
  });
});
