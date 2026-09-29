import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SetupWorkspaceStep } from "@/components/auth/SetupWorkspaceStep";

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
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/workspaces") return { data: [] };
      throw new Error(`unexpected GET ${url}`);
    });
  });

  it("shows the error when the workspaces cannot load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    renderStep();

    expect(await screen.findByText(/something went wrong/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/workspace name/i)).not.toBeInTheDocument();
  });

  it("blocks an empty workspace name instead of continuing", async () => {
    const user = userEvent.setup();
    const onContinue = renderStep();

    await user.clear(await screen.findByLabelText(/workspace name/i));
    await user.click(screen.getByRole("button", { name: /continue/i }));

    expect(await screen.findByText(/workspace name is required/i)).toBeInTheDocument();
    expect(onContinue).not.toHaveBeenCalled();
  });

  it("asks for the workspace name only, with no project fields", async () => {
    renderStep();

    expect(await screen.findByLabelText(/workspace name/i)).toHaveValue("Default");
    expect(screen.queryByLabelText(/project/i)).not.toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalledWith("/api/projects", expect.anything());
  });

  it("hands the name up instead of saving it itself", async () => {
    // Renaming needs workspaces:write, which the owner only holds once the wizard completes.
    const user = userEvent.setup();
    const onContinue = renderStep();

    const name = await screen.findByLabelText(/workspace name/i);
    await user.clear(name);
    await user.type(name, "Acme");
    await user.click(screen.getByRole("button", { name: /continue/i }));

    expect(mocks.patch).not.toHaveBeenCalled();
    expect(mocks.post).not.toHaveBeenCalled();
    expect(onContinue).toHaveBeenCalledWith({ workspaceId: "workspace-default", workspaceName: "Acme" });
  });
});
