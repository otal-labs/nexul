import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectDesktopCard } from "@/components/you/ConnectDesktopCard";

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderCard = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ConnectDesktopCard />
    </QueryClientProvider>,
  );
};

// user-event installs its own clipboard stub in setup(), so ours has to go on after it.
const setupUser = (writeText: ReturnType<typeof vi.fn>) => {
  const user = userEvent.setup();
  Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true, writable: true });
  return user;
};

describe("ConnectDesktopCard", () => {
  const writeText = vi.fn();

  beforeEach(() => {
    mocks.post.mockReset();
    mocks.errorMessage.mockReset();
    writeText.mockReset().mockResolvedValue(undefined);
    vi.mocked(toast.error).mockClear();
  });

  it("mints a token, copies it, and shows the tick", async () => {
    mocks.post.mockResolvedValue({ data: { token: "header.payload.sig" } });
    const user = setupUser(writeText);
    renderCard();

    await user.click(screen.getByRole("button", { name: "Copy connection token" }));

    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/connection-token");
    expect(writeText).toHaveBeenCalledWith("header.payload.sig");
  });

  it("shows an error toast and keeps the label when minting fails", async () => {
    mocks.post.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Could not mint");
    const user = setupUser(writeText);
    renderCard();

    await user.click(screen.getByRole("button", { name: "Copy connection token" }));

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Could not mint"));
    expect(screen.getByRole("button", { name: "Copy connection token" })).toBeEnabled();
    expect(writeText).not.toHaveBeenCalled();
  });

  it("shows an error toast when the clipboard refuses the token", async () => {
    mocks.post.mockResolvedValue({ data: { token: "header.payload.sig" } });
    writeText.mockRejectedValue(new Error("denied"));
    mocks.errorMessage.mockReturnValue("Clipboard unavailable");
    const user = setupUser(writeText);
    renderCard();

    await user.click(screen.getByRole("button", { name: "Copy connection token" }));

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Clipboard unavailable"));
    expect(screen.queryByRole("button", { name: "Copied" })).not.toBeInTheDocument();
  });
});
