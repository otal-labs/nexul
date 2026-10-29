import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnerMenu } from "@/components/runner/RunnerMenu";
import { resetViewer, viewer } from "@/test/viewerActions";

const mocks = vi.hoisted(() => ({ delete: vi.fn(), confirm: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { delete: mocks.delete }, errorMessage: () => "Something went wrong" }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));

const runner = {
  id: "r-1",
  name: "alpha",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  running_job: null,
  version: "v0.1.4",
};

const manageable = { actions: ["machines:see", "machines:write"] };

const renderButton = (connected: boolean, machine: { actions: string[] } = manageable) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RunnerMenu runner={{ ...runner, connected }} machine={machine} />
    </QueryClientProvider>,
  );

const remove = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button", { name: "More actions for alpha" }));
  return screen.findByRole("menuitem", { name: /Remove runner/ });
};

describe("RunnerMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetViewer();
    viewer.showLocked = true;
    mocks.delete.mockResolvedValue({});
  });

  it("does nothing when the removal is not confirmed", async () => {
    mocks.confirm.mockResolvedValue(false);
    const user = userEvent.setup();
    renderButton(true);

    await user.click(await remove(user));

    expect(mocks.confirm).toHaveBeenCalledWith(expect.objectContaining({ title: "Remove alpha?", confirmLabel: "Remove runner" }));
    expect(mocks.delete).not.toHaveBeenCalled();
  });

  it("says an offline runner uninstalls itself when it returns", async () => {
    mocks.confirm.mockResolvedValue(false);
    const user = userEvent.setup();
    renderButton(false);

    await user.click(await remove(user));

    expect(mocks.confirm).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining("next time it comes online") }));
  });

  it("removes the runner once confirmed", async () => {
    mocks.confirm.mockResolvedValue(true);
    const user = userEvent.setup();
    renderButton(true);

    await user.click(await remove(user));

    await waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/runners/r-1"));
  });

  it("is locked on a machine the viewer only sees, and removes nothing", async () => {
    const user = userEvent.setup();
    renderButton(true, { actions: ["machines:see"] });

    const item = await remove(user);
    await user.click(item);

    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(item).toHaveTextContent("Your role can't remove runners");
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(mocks.delete).not.toHaveBeenCalled();
  });

  it("draws no ⋯ for a viewer whose roles don't show locked actions", () => {
    viewer.showLocked = false;
    renderButton(true, { actions: ["machines:see"] });

    expect(screen.queryByRole("button", { name: "More actions for alpha" })).not.toBeInTheDocument();
  });
});
