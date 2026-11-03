import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CloneRepositoryButton } from "@/components/settings/CloneRepositoryButton";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", async (original) => ({
  ...(await original<typeof import("@/api/client")>()),
  api: { get: mocks.get, post: mocks.post },
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const winget = ["winget", "install", "--id", "Git.Git", "--exact", "--source", "winget"];

const refused = (details?: object) => ({
  response: { data: { message: "git isn't installed on Desk; install it from the computer's row in Settings → Computers", code: "CONFLICT", details } },
});

const renderButton = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <CloneRepositoryButton computerId="c1" repository="acme/api" onCloned={() => {}} />
    </QueryClientProvider>,
  );
};

describe("CloneRepositoryButton", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    vi.mocked(toast.error).mockClear();
    mocks.get.mockResolvedValue({ data: { computers: [{ id: "c1", name: "Desk", facts: { t3: { state: "answering" }, tools: [{ name: "git", install: winget }], providers: [], projects: [] } }] } });
  });

  it("names the missing git when a clone fails for want of it, and installs it from the toast after showing the command", async () => {
    mocks.post.mockRejectedValueOnce(refused({ missing_tool: "git", computer_id: "c1" })).mockResolvedValueOnce({ data: undefined });
    const user = userEvent.setup();
    renderButton();

    await user.click(screen.getByRole("button", { name: "Clone acme/api" }));
    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1));
    const [message, options] = vi.mocked(toast.error).mock.calls[0] ?? [];
    expect(message).toBe("git isn't installed on Desk; install it from the computer's row in Settings → Computers");
    const action = (options as { action: { label: string; onClick: () => void } }).action;
    expect(action.label).toBe("Install git");

    await act(async () => action.onClick());
    expect(await screen.findByText("Install git on Desk?")).toBeInTheDocument();
    expect(screen.getByText(winget.join(" "))).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "Install git" }));
    await waitFor(() => expect(mocks.post).toHaveBeenLastCalledWith("/api/pairing/computers/c1/tools", { tool: "git" }));
  });

  it("toasts any other failed clone once, with no install offer", async () => {
    mocks.post.mockRejectedValueOnce(refused());
    const user = userEvent.setup();
    renderButton();

    await user.click(screen.getByRole("button", { name: "Clone acme/api" }));

    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1));
    expect(vi.mocked(toast.error).mock.calls[0]).toHaveLength(1);
  });
});
