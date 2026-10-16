import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayForm } from "@/components/play/PlayForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SavePlayFormSchema, type Play, type SavePlayFormData } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";

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
  builtin_key: "",
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

// The viewer's role-wide permissions, the play's own answers (none set), and what a rename would break.
const serve = (impact: { count: number; automations: string[] } | undefined = undefined) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["plays:write"], show_locked_actions: true } };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    return { data: url.endsWith("/rename-impact") && impact ? impact : [] };
  });

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  serve();
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

    expect(await screen.findByText(/needs one show-when stage/)).toBeInTheDocument();
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

  it("creates a stack play with no show-when stage", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: { id: "play-1" } });
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Label"), "Investigate a crash");
    await user.click(screen.getByLabelText("Type"));
    await user.click(screen.getByRole("option", { name: "Stack" }));
    expect(screen.queryByLabelText("Show when")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Create play" }));
    expect(api.post).toHaveBeenCalledWith(
      "/api/workspaces/ws-1/plays",
      expect.objectContaining({ type: "stack", show_when_stage: null }),
    );
  });

  it("shows a taken label on the label field", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockRejectedValue({
      response: {
        status: 409,
        data: { message: "taken", code: "CONFLICT", errors: { label: ['conflict: a play named "Fix with AI" already exists in this workspace'] } },
      },
    });
    renderHarness();

    await user.click(screen.getByRole("button", { name: "Open" }));
    await user.type(screen.getByLabelText("Label"), "Fix with AI");
    await user.click(screen.getByRole("button", { name: "Create play" }));

    expect(await screen.findByText('a play named "Fix with AI" already exists in this workspace')).toBeInTheDocument();
    expect(screen.getByLabelText("Label")).toHaveAttribute("aria-invalid", "true");
    expect(toast.error).not.toHaveBeenCalled();
    expect(screen.getByText("pending")).toBeInTheDocument();
    await user.keyboard("{Escape}");
  });

  it("shows type as read-only text when editing", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: { id: "play-fix" } });
    renderHarness(editingPlay);

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.queryByLabelText("Type")).not.toBeInTheDocument();
    expect(screen.getByText(/can't change once created/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(api.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix", expect.objectContaining({
      label: "Fix with AI",
    }));
  });

  it("holds Save and shows the sentence when a rule on this play denies editing it", async () => {
    const user = userEvent.setup();
    const sentence = "Denied on play Fix with AI for role Member";
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["plays:write"], show_locked_actions: true } };
      if (url === "/api/permissions/explain") return { data: { effective: [{ action: "plays:write", allowed: false, source: { sentence } }] } };
      return { data: [] };
    });
    renderHarness(editingPlay);

    await user.click(screen.getByRole("button", { name: "Open" }));

    expect(await screen.findByText(sentence)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  describe("renaming", () => {
    const impactOf = serve;

    const rename = async (user: ReturnType<typeof userEvent.setup>) => {
      await user.click(screen.getByRole("button", { name: "Open" }));
      await user.clear(screen.getByLabelText("Label"));
      await user.type(screen.getByLabelText("Label"), "Fix it");
      await user.click(screen.getByRole("button", { name: "Save" }));
    };

    it("asks first, naming the automations that call the old name, and Cancel keeps the old label", async () => {
      const user = userEvent.setup();
      impactOf({ count: 2, automations: ["Deploy notifier", "Nightly triage"] });
      renderHarness(editingPlay);

      await rename(user);

      expect(
        await screen.findByText(
          "2 automations call this play by its old name: Deploy notifier, Nightly triage. They'll fail until their code uses the new name.",
        ),
      ).toBeInTheDocument();
      expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix/rename-impact", { params: { label: "Fix it" } });
      await user.click(screen.getByRole("button", { name: "Cancel" }));

      expect(api.patch).not.toHaveBeenCalled();
      expect(screen.getByLabelText("Label")).toHaveValue("Fix with AI");
      expect(screen.getByText("pending")).toBeInTheDocument();
      await user.keyboard("{Escape}");
    });

    it("renames once the person confirms", async () => {
      const user = userEvent.setup();
      impactOf({ count: 1, automations: [] });
      vi.mocked(api.patch).mockResolvedValue({ data: { id: "play-fix" } });
      renderHarness(editingPlay);

      await rename(user);
      expect(await screen.findByText("1 automation calls this play by its old name. It'll fail until its code uses the new name.")).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: "Rename anyway" }));

      expect(await screen.findByText("saved")).toBeInTheDocument();
      expect(api.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix", expect.objectContaining({ label: "Fix it" }));
    });

    it("saves without asking when no automation calls the old name", async () => {
      const user = userEvent.setup();
      impactOf({ count: 0, automations: [] });
      vi.mocked(api.patch).mockResolvedValue({ data: { id: "play-fix" } });
      renderHarness(editingPlay);

      await rename(user);

      expect(await screen.findByText("saved")).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Rename anyway" })).not.toBeInTheDocument();
    });
  });
});
