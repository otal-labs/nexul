import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/WorkspaceHooks", () => ({ useHasPermission: () => true }));

const template = {
  workspace_id: "ws-1",
  body: "## Mine",
  default_body: "## Stack and versions",
  updated_by: "u-1",
  updated_at: "2026-09-24T12:00:00Z",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
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

  it("resets to the seeded categories", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Reset to default" }));
    expect(screen.getByLabelText("Template (markdown)")).toHaveValue("## Stack and versions");
    expect(screen.getByRole("button", { name: "Reset to default" })).toBeDisabled();
  });

  it("warns once the template is over the cap", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { ...template, body: "x".repeat(8001) } });
    renderSection();
    expect(await screen.findByRole("alert")).toHaveTextContent("over the cap");
  });
});
