import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayForm } from "@/components/play/PlayForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SavePlayFormSchema, type Play, type SavePlayFormData } from "@/models/Play";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const editingPlay: Play = {
  id: "play-fix",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  created_by: "",
  created_at: "",
  updated_at: "",
};

const PlayHarness = ({ editing }: { editing?: Play }) => {
  const { open } = useFormDialog();
  const [result, setResult] = useState("pending");
  const defaults: SavePlayFormData = editing
    ? {
        label: editing.label,
        type: editing.type,
        description: editing.description,
        instructions: editing.instructions,
        enabled: editing.enabled,
        show_when_stage: editing.show_when_stage ?? "",
        excluded_project_ids: editing.excluded_project_ids,
      }
    : {
        label: "",
        type: "ticket",
        description: "",
        instructions: "",
        enabled: true,
        show_when_stage: "progress",
        excluded_project_ids: [],
      };
  return (
    <div>
      <button
        type="button"
        onClick={async () => {
          const outcome = await open<SavePlayFormData>({
            title: editing ? "Edit play" : "New play",
            schema: SavePlayFormSchema,
            okLabel: editing ? "Save" : "Create play",
            form: <PlayForm workspaceId="ws-1" {...(editing ? { editing } : {})} />,
            formOptions: { defaultValues: defaults },
          });
          setResult(outcome.success ? "saved" : "cancelled");
        }}
      >
        Open
      </button>
      <span>{result}</span>
    </div>
  );
};

const renderHarness = (editing?: Play) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <PlayHarness {...(editing ? { editing } : {})} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: [] });
});

describe("PlayForm", () => {
  it("creates a ticket play with its show-when stage", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "play-1" } });
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Label"), "Fix with AI");
    await user.click(screen.getByRole("button", { name: "Create play" }));

    expect(api.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays", expect.objectContaining({
      label: "Fix with AI",
      type: "ticket",
      show_when_stage: "progress",
    }));
  });

  it("rejects an empty label", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.click(screen.getByRole("button", { name: "Create play" }));

    expect(await screen.findByText("Label is required")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.keyboard("{Escape}");
  });

  it("rejects a ticket play with no show-when stage", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Label"), "Fix with AI");
    await user.click(screen.getByLabelText("Show when"));
    await user.click(screen.getByRole("option", { name: "No stage" }));
    await user.click(screen.getByRole("button", { name: "Create play" }));

    expect(await screen.findByText(/needs exactly one show-when stage/)).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.keyboard("{Escape}");
  });

  it("does not show a show-when field for a doc play", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "play-1" } });
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Label"), "To tickets via AI");
    await user.click(screen.getByLabelText("Type"));
    await user.click(screen.getByRole("option", { name: "Doc" }));
    expect(screen.queryByLabelText("Show when")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Create play" }));
    expect(api.post).toHaveBeenCalledWith(
      "/api/workspaces/ws-1/plays",
      expect.objectContaining({ type: "doc", show_when_stage: null }),
    );
  });

  it("shows type as read-only text when editing", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: { id: "play-fix" } });
    renderHarness(editingPlay);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.queryByLabelText("Type")).not.toBeInTheDocument();
    expect(screen.getByText(/can't change after create/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(api.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix", expect.objectContaining({
      label: "Fix with AI",
    }));
  });
});
