import type { ReactElement } from "react";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { CreateDocForm } from "@/components/doc/CreateDocForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveDocFormSchema, type SaveDocFormData } from "@/models/Doc";
import { emptyDocForm } from "@/utils/emptyDocJson";
import { emptyDocJson, isStructuredBody } from "@/utils/RichtextUtility";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const projects = [
  { id: "p-1", name: "Backend", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", position: 1, created_at: "", updated_at: "" },
];

const mockProjects = (list: unknown[] = projects) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url.startsWith("/api/projects")) return { data: list };
    return { data: [] };
  });
};

const DocHarness = () => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const result = await open<SaveDocFormData>({
            title: "New doc",
            schema: SaveDocFormSchema,
            okLabel: "Create",
            form: <CreateDocForm />,
            formOptions: { defaultValues: emptyDocForm() },
          });
          setResult(result.success ? String((result.data as SaveDocFormData & { id?: string })?.id) : "cancelled");
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
});

describe("CreateDocForm", () => {
  it("creates a doc in the selected project and resolves with its id", async () => {
    const user = userEvent.setup();
    mockProjects();
    vi.mocked(api.post).mockResolvedValue({
      data: {
        id: "doc-1",
        project_id: "p-1",
        title: "New doc",
        body: "",
        version: 1,
        archived: false,
        created_at: "",
        updated_at: "",
      },
    });
    renderWithRoot(<DocHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "New doc");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("doc-1")).toBeInTheDocument();
    expect(api.post).toHaveBeenCalledWith(
      "/api/docs",
      expect.objectContaining({ project_id: "p-1", title: "New doc" }),
    );
  });

  it("submits a structured rich-text body", async () => {
    const user = userEvent.setup();
    mockProjects();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "doc-1" } });
    renderWithRoot(<DocHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(await screen.findByLabelText("Title"), "New doc");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await vi.waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
    const [, payload] = vi.mocked(api.post).mock.calls[0] as [string, { title: string; body: string }];
    expect(payload.body).toBe(emptyDocJson);
    expect(isStructuredBody(payload.body)).toBe(true);
  });

  it("renders the body editor with an accessible region", async () => {
    const user = userEvent.setup();
    mockProjects();
    renderWithRoot(<DocHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByLabelText("Body")).toBeInTheDocument();
    expect(screen.getByText("Start writing…")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("rejects an empty title without submitting", async () => {
    const user = userEvent.setup();
    mockProjects();
    renderWithRoot(<DocHarness />);

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
    mockProjects([]);
    renderWithRoot(<DocHarness />);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(await screen.findByText(/create a project first/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
    await user.keyboard("{Escape}");
  });
});
