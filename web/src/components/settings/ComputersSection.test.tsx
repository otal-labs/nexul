import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputersSection } from "@/components/settings/ComputersSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));

const computerList = (computers: unknown[]) => ({ computers });

const computer = (overrides: Record<string, unknown> = {}) => ({
  id: "c1",
  name: "Home",
  server_url: "https://home.example.com",
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.0.34",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ComputersSection />
    </QueryClientProvider>,
  );
};

describe("ComputersSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockResolvedValue({ data: computerList([]) });
  });

  it("shows an empty state when there are no computers", async () => {
    renderSection();
    expect(await screen.findByText(/no computers paired yet/i)).toBeInTheDocument();
  });

  it("lists paired computers", async () => {
    mocks.get.mockResolvedValue({ data: computerList([computer()]) });
    renderSection();

    expect(await screen.findByText("Home")).toBeInTheDocument();
    expect(screen.getByText(/https:\/\/home\.example\.com.*0\.0\.34/)).toBeInTheDocument();
  });

  it("flags a computer expiring within the warning window", async () => {
    const soon = new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString();
    mocks.get.mockResolvedValue({ data: computerList([computer({ token_expires_at: soon })]) });
    renderSection();

    expect(await screen.findByText(/expires in \dd/)).toBeInTheDocument();
  });

  it("flags an expired computer as acting like unpaired", async () => {
    const past = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    mocks.get.mockResolvedValue({ data: computerList([computer({ token_expires_at: past })]) });
    renderSection();

    expect(await screen.findByText(/acts as unpaired/i)).toBeInTheDocument();
  });

  it("reads a computer tunnel with no session yet as pairing in progress, not expired", async () => {
    mocks.get.mockResolvedValue({
      data: computerList([
        computer({
          server_url: "https://laptop-ab12cd34.example.com",
          token_expires_at: "0001-01-01T00:00:00Z",
          harness_version: "",
          tunnel: { tunnel_id: "tun-1", hostname: "laptop-ab12cd34.example.com" },
        }),
      ]),
    });
    renderSection();

    expect(await screen.findByText(/pairing in progress/i)).toBeInTheDocument();
    expect(screen.queryByText(/acts as unpaired/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^pair$/i })).toBeInTheDocument();
  });

  it("has no separate Pair by URL button; URL pairing lives in the dialog", async () => {
    renderSection();
    await screen.findByText(/no computers paired yet/i);
    expect(screen.queryByRole("button", { name: /pair by url/i })).not.toBeInTheDocument();
  });

  it("opens the pair-a-computer dialog on its Connect step", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /pair a computer/i }));
    expect(await screen.findByRole("dialog", { name: /pair a computer/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/computer name/i)).toBeInTheDocument();
  });

  it("re-pairs an existing computer, pre-filled with its name and URL", async () => {
    mocks.get.mockResolvedValue({ data: computerList([computer()]) });
    mocks.post.mockResolvedValue({ data: computer({ harness_version: "0.0.35" }) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /re-pair/i }));
    const nameInput = screen.getByLabelText(/^name$/i) as HTMLInputElement;
    const urlInput = screen.getByLabelText(/t3 server url/i) as HTMLInputElement;
    expect(nameInput.value).toBe("Home");
    expect(urlInput.value).toBe("https://home.example.com");

    await user.type(screen.getByLabelText(/one-time pairing token/i), "fresh-tok");
    await user.click(screen.getByRole("button", { name: /^re-pair$/i }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/repair", {
        name: "Home",
        server_url: "https://home.example.com",
        token: "fresh-tok",
      }),
    );
  });

  it("removes a computer after arming the confirm step", async () => {
    mocks.get.mockResolvedValue({ data: computerList([computer()]) });
    mocks.del.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^remove$/i }));
    expect(mocks.del).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /^confirm$/i }));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/c1"));
  });
});
