import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DeployCancelButton } from "@/components/deploy/DeployCancelButton";
import { resetViewer, viewer } from "@/test/viewerActions";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", async () => {
  const { withOwner } = await import("@/test/viewer");
  return { api: { get: withOwner(mocks.get), post: mocks.post }, errorMessage: () => "boom" };
});
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

const deploy = { id: "d-1", stack_id: "stack-1" };

const renderButton = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <DeployCancelButton deploy={deploy} />
    </QueryClientProvider>,
  );

describe("DeployCancelButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetViewer();
    viewer.workspace = ["deploys:write"];
    mocks.get.mockResolvedValue({ data: { id: "stack-1", project_id: "proj-1", name: "api" } });
    mocks.post.mockResolvedValue({ data: {} });
  });

  it("cancels the deploy for someone who can deploy", async () => {
    renderButton();
    await userEvent.setup().click(await screen.findByRole("button", { name: "Cancel deploy" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/deploys/d-1/cancel");
  });

  it("locks Cancel with the reason for someone who can't, and cancels nothing", async () => {
    viewer.workspace = [];
    viewer.showLocked = true;
    const user = userEvent.setup();
    renderButton();

    const cancel = await screen.findByRole("button", { name: "Cancel deploy" });
    await user.click(cancel);

    expect(cancel).toHaveAttribute("aria-disabled", "true");
    expect(cancel).toHaveAccessibleDescription("Your role can't cancel deploys");
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
