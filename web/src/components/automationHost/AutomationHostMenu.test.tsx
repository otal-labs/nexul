import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationHostMenu } from "@/components/automationHost/AutomationHostMenu";
import { resetViewer, viewer } from "@/test/viewerActions";

const mocks = vi.hoisted(() => ({ delete: vi.fn(), confirm: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { delete: mocks.delete }, errorMessage: () => "Something went wrong" }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);
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
  managers: [],
};

const renderButton = (connected: boolean, actions: string[] = ["machines:see", "machines:write"]) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <AutomationHostMenu host={{ ...host, connected }} machine={{ actions }} />
    </QueryClientProvider>,
  );

const remove = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button", { name: "More actions for jobs-1" }));
  return screen.findByRole("menuitem", { name: /Remove host/ });
};

describe("AutomationHostMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetViewer();
    mocks.delete.mockResolvedValue({});
  });

  it("does nothing when the removal is not confirmed", async () => {
    mocks.confirm.mockResolvedValue(false);
    const user = userEvent.setup();
    renderButton(true);

    await user.click(await remove(user));

    expect(mocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Remove jobs-1?", confirmLabel: "Remove automations host" }),
    );
    expect(mocks.delete).not.toHaveBeenCalled();
  });

  it("says an offline host uninstalls itself when it returns, and where its automations go", async () => {
    mocks.confirm.mockResolvedValue(false);
    const user = userEvent.setup();
    renderButton(false);

    await user.click(await remove(user));

    expect(mocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({ message: expect.stringMatching(/next time it comes online.*move to the instance host/) }),
    );
  });

  it("removes the host once confirmed", async () => {
    mocks.confirm.mockResolvedValue(true);
    const user = userEvent.setup();
    renderButton(true);

    await user.click(await remove(user));

    await waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/automation-hosts/h-1"));
  });

  it("is hidden from someone who only sees or deploys to the machine", () => {
    renderButton(true, ["machines:see", "machines:deploy"]);

    expect(screen.queryByRole("button", { name: "More actions for jobs-1" })).not.toBeInTheDocument();
  });

  it("is locked, with the reason, for a role that shows locked actions", async () => {
    viewer.showLocked = true;
    const user = userEvent.setup();
    renderButton(true, ["machines:see"]);

    const item = await remove(user);
    await user.click(item);

    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(item).toHaveTextContent("Your role can't remove automations hosts");
    expect(mocks.confirm).not.toHaveBeenCalled();
  });

  it("answers to the instance Owner for a host on a machine Nexul doesn't know", async () => {
    viewer.instanceOwner = true;
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <AutomationHostMenu host={host} />
      </QueryClientProvider>,
    );

    expect(await remove(user)).not.toHaveAttribute("aria-disabled");
  });
});
