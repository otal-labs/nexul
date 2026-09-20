import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SetupWorkspaceStep } from "@/components/auth/SetupWorkspaceStep";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, patch: mocks.patch },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const workspace = { id: "ws-1", name: "Acme", created_at: "", updated_at: "" };
const project = { id: "p-1", name: "General", prefix: "", position: 0, created_at: "", updated_at: "" };

const renderStep = (onContinue = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <SetupWorkspaceStep onContinue={onContinue} />
    </QueryClientProvider>,
  );
  return onContinue;
};

describe("SetupWorkspaceStep", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.errorMessage.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/workspaces") return { data: [workspace] };
      if (url === "/api/projects") return { data: [project] };
      throw new Error(`unexpected GET ${url}`);
    });
  });

  it("renders the workspace name and project fields", async () => {
    renderStep();
    expect(await screen.findByLabelText(/project name/i)).toHaveValue("General");
    expect(screen.getByLabelText(/project prefix/i)).toHaveValue("");
    expect(screen.getByLabelText(/workspace name/i)).toHaveValue("Acme");
  });

  it("hands the validated project name and prefix up instead of saving them itself", async () => {
    // The PATCH endpoints require can_create_workspace, which this user only
    // gets from CompleteOwnerWizard (runs after this step), so it must not
    // call the backend itself.
    const user = userEvent.setup();
    const onContinue = renderStep();

    await screen.findByLabelText(/project name/i);
    await user.clear(screen.getByLabelText(/project name/i));
    await user.type(screen.getByLabelText(/project name/i), "Nexul");
    await user.type(screen.getByLabelText(/project prefix/i), "DEP");
    await user.click(screen.getByRole("button", { name: /continue/i }));

    expect(mocks.patch).not.toHaveBeenCalled();
    expect(mocks.post).not.toHaveBeenCalled();
    expect(onContinue).toHaveBeenCalledWith({
      workspaceId: "workspace-default",
      workspaceName: "Acme",
      projectId: "p-1",
      projectName: "Nexul",
      projectPrefix: "DEP",
    });
  });

  it("blocks submit on an invalid prefix instead of continuing", async () => {
    const user = userEvent.setup();
    const onContinue = renderStep();

    await screen.findByLabelText(/project name/i);
    await user.type(screen.getByLabelText(/project prefix/i), "1");
    await user.click(screen.getByRole("button", { name: /continue/i }));

    expect(await screen.findByText(/prefix must be 2-5 letters/i)).toBeInTheDocument();
    expect(onContinue).not.toHaveBeenCalled();
  });
});
