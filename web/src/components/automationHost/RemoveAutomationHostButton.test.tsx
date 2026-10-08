import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RemoveAutomationHostButton } from "@/components/automationHost/RemoveAutomationHostButton";

const mocks = vi.hoisted(() => ({ delete: vi.fn(), confirm: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { delete: mocks.delete }, errorMessage: () => "Something went wrong" }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));

const host = {
  id: "h-1",
  name: "jobs-1",
  machine: "prod",
  os: "linux",
  arch: "amd64",
  version: "v0.3.0",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
};

const renderButton = (connected: boolean) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RemoveAutomationHostButton host={{ ...host, connected }} />
    </QueryClientProvider>,
  );

describe("RemoveAutomationHostButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.delete.mockResolvedValue({});
  });

  it("does nothing when the removal is not confirmed", async () => {
    mocks.confirm.mockResolvedValue(false);
    const user = userEvent.setup();
    renderButton(true);

    await user.click(screen.getByRole("button", { name: "Remove jobs-1" }));

    expect(mocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Remove jobs-1?", confirmLabel: "Remove automations host" }),
    );
    expect(mocks.delete).not.toHaveBeenCalled();
  });

  it("says an offline host uninstalls itself when it returns, and where its automations go", async () => {
    mocks.confirm.mockResolvedValue(false);
    const user = userEvent.setup();
    renderButton(false);

    await user.click(screen.getByRole("button", { name: "Remove jobs-1" }));

    expect(mocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({ message: expect.stringMatching(/next time it comes online.*move to the instance host/) }),
    );
  });

  it("removes the host once confirmed", async () => {
    mocks.confirm.mockResolvedValue(true);
    const user = userEvent.setup();
    renderButton(true);

    await user.click(screen.getByRole("button", { name: "Remove jobs-1" }));

    await waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/automation-hosts/h-1"));
  });
});
