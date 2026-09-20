import type { ReactElement } from "react";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { CreateMemoryForm } from "@/components/memory/CreateMemoryForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { CreateMemoryFormSchema, emptyCreateMemoryForm, type CreateMemoryFormData } from "@/models/Memory";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, icon: "", created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, icon: "", created_at: "", updated_at: "" },
];

const mockProjects = (list: unknown[] = projects) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url.startsWith("/api/projects")) return { data: list };
    return { data: [] };
  });
};

const MemoryHarness = ({ defaultProjectId }: { defaultProjectId?: string }) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<CreateMemoryFormData>({
            title: "New memory",
            schema: CreateMemoryFormSchema,
            okLabel: "Create",
            form: <CreateMemoryForm {...(defaultProjectId ? { defaultProjectId } : {})} />,
            formOptions: { defaultValues: emptyCreateMemoryForm() },
          });
          setResult(result.success ? String((result.data as CreateMemoryFormData & { id?: string })?.id) : "cancelled");
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

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("CreateMemoryForm", () => {
  it("defaults the project select to Workspace and sends the workspace id", async () => {
    const user = userEvent.setup();
    mockProjects();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "mem-1" } });
    renderWithRoot(<MemoryHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByRole("combobox", { name: "Project" })).toHaveTextContent("Workspace");
    await user.type(screen.getByLabelText("Title"), "New memory");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("mem-1")).toBeInTheDocument();
    expect(api.post).toHaveBeenCalledWith(
      "/api/memories",
      expect.objectContaining({ project_id: "", workspace_id: "ws-1", title: "New memory" }),
    );
  });

  it("defaults to the given project instead of Workspace when one is passed", async () => {
    const user = userEvent.setup();
    mockProjects();
    renderWithRoot(<MemoryHarness defaultProjectId="p-1" />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByRole("combobox", { name: "Project" })).toHaveTextContent("Backend");
    await user.keyboard("{Escape}");
  });

  it("offers Workspace with no other options when there are no projects", async () => {
    const user = userEvent.setup();
    mockProjects([]);
    renderWithRoot(<MemoryHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(await screen.findByRole("combobox", { name: "Project" }));
    expect(await screen.findByRole("option", { name: "Workspace" })).toBeInTheDocument();
    expect(screen.getAllByRole("option")).toHaveLength(1);
    await user.keyboard("{Escape}");

    expect(screen.queryByText(/create a project first/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).not.toBeDisabled();
    await user.keyboard("{Escape}");
  });

  it("creates a project-scoped memory when a project is picked", async () => {
    const user = userEvent.setup();
    mockProjects();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "mem-2" } });
    renderWithRoot(<MemoryHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(await screen.findByRole("combobox", { name: "Project" }));
    await user.click(await screen.findByRole("option", { name: "Frontend" }));
    await user.type(screen.getByLabelText("Title"), "New memory");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("mem-2")).toBeInTheDocument();
    expect(api.post).toHaveBeenCalledWith(
      "/api/memories",
      expect.objectContaining({ project_id: "p-2", workspace_id: "ws-1" }),
    );
  });
});
