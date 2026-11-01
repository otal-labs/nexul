import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PasskeysSection } from "@/components/you/PasskeysSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
  create: vi.fn(),
  supported: vi.fn(() => true),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.del },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}));

vi.mock("@/lib/passkeys", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/passkeys")>()),
  passkeysSupported: mocks.supported,
  createPasskey: mocks.create,
}));

vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));

const laptop = { id: "cred-1", name: "Firefox on Linux", rp_id: window.location.hostname, created_at: "2026-10-01T00:00:00Z" };

const serve = (passkeys: unknown[], available = true) =>
  mocks.get.mockResolvedValue({ data: { passkeys, available } });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PasskeysSection />
    </QueryClientProvider>,
  );
};

describe("PasskeysSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.supported.mockReturnValue(true);
  });

  it("adds a passkey by handing the browser's credential back with its ceremony", async () => {
    serve([]);
    mocks.post.mockImplementation((url: string) => {
      if (url === "/api/auth/passkeys/start") return Promise.resolve({ data: { ceremony: "pkc_1", options: { challenge: "abc" } } });
      return Promise.resolve({ data: laptop });
    });
    mocks.create.mockResolvedValue({ id: "cred-1" });
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Add a passkey" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/auth/passkeys", { ceremony: "pkc_1", credential: { id: "cred-1" } }));
    expect(mocks.create).toHaveBeenCalledWith({ challenge: "abc" });
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Passkey added: Firefox on Linux");
  });

  it("says nothing when the person closes the browser's prompt", async () => {
    serve([]);
    mocks.post.mockResolvedValue({ data: { ceremony: "pkc_1", options: {} } });
    mocks.create.mockRejectedValue(new DOMException("closed", "NotAllowedError"));
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Add a passkey" }));

    await waitFor(() => expect(mocks.create).toHaveBeenCalled());
    expect(mocks.toastError).not.toHaveBeenCalled();
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("offers no Add off https and says why", async () => {
    serve([], false);
    renderSection();

    expect(await screen.findByText("Passkeys need Nexul on an https address.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add a passkey" })).not.toBeInTheDocument();
  });

  it("marks a passkey made for another domain, which can't sign in here", async () => {
    serve([{ ...laptop, rp_id: "old.example.com" }]);
    renderSection();

    expect(await screen.findByText("Made for another domain")).toBeInTheDocument();
    expect(screen.getByText("old.example.com")).toBeInTheDocument();
  });

  it("removes a passkey after the confirm", async () => {
    serve([laptop]);
    mocks.del.mockResolvedValue({});
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Remove Firefox on Linux" }));
    await userEvent.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/auth/passkeys/cred-1"));
  });
});
