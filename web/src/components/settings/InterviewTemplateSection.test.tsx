import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api, errorMessage } from "@/api/client";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/WorkspaceHooks", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/WorkspaceHooks")>()),
  useHasPermission: () => true,
}));

const template = {
  workspace_id: "ws-1",
  body: "## Mine",
  questions: [{ text: "Mine", hint: "", multi_select: false, options: [] }],
  default_body: "## What languages and frameworks does this project use?",
  edited: true,
  updated_by: "u-1",
  updated_at: "2026-09-24T12:00:00Z",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <InterviewTemplateSection />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: template });
});

describe("InterviewTemplateSection", () => {
  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderSection();
    expect(await screen.findByText("Something went wrong")).toBeInTheDocument();
  });

  it("saves an edited template", async () => {
    const user = userEvent.setup();
    vi.mocked(api.put).mockResolvedValue({ data: { ...template, body: "## Mine!" } });
    renderSection();

    const field = await screen.findByLabelText("Template (markdown)");
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    await user.type(field, "!");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(api.put).toHaveBeenCalledWith("/api/memories/interview-template", { workspace_id: "ws-1", body: "## Mine!" });
  });

  it("says it follows the instance template while the workspace has not edited its own", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { ...template, body: "## Instance", default_body: "## Instance", edited: false } });
    renderSection();

    expect(await screen.findByText("Following the instance template")).toBeInTheDocument();
    expect(screen.getByLabelText("Template (markdown)")).toHaveValue("## Instance");
    expect(screen.queryByRole("button", { name: "Reset to instance template" })).not.toBeInTheDocument();
  });

  it("resets an edited workspace template to the instance's after a confirm", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    renderSection();

    expect(await screen.findByText("Edited for this workspace")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Reset to instance template" }));
    await user.click(await screen.findByRole("button", { name: "Reset" }));
    expect(api.post).toHaveBeenCalledWith("/api/templates/reset", {
      kind: "interview",
      key: "",
      at: { scope: "workspace", workspace_id: "ws-1" },
    });
  });

  it("counts the saved template's questions", async () => {
    renderSection();
    expect(await screen.findByText("1 question")).toBeInTheDocument();
  });

  it("shows a refused save inline, naming the line", async () => {
    const user = userEvent.setup();
    vi.mocked(api.put).mockRejectedValue(new Error("refused"));
    vi.mocked(errorMessage).mockReturnValue("line 1: text before the first ## question heading");
    renderSection();

    await user.type(await screen.findByLabelText("Template (markdown)"), "!");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("line 1: text before the first ## question heading");
  });
});
