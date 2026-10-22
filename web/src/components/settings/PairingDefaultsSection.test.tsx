import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PairingDefaultsSection } from "@/components/settings/PairingDefaultsSection";
import { pickOption } from "@/test/pickOption";

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

const computer = (id: string, name: string) => ({
  id,
  name,
  server_url: `https://${name}.example.com`,
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.0.34",
  created_at: "2026-08-01T00:00:00Z",
  position: 0,
  runs_at_once: 1,
  running: 0,
  chat_project_id: "",
  updated_at: "2026-08-01T00:00:00Z",
});

const mockGet = (url: string) => {
  if (url === "/api/pairing/computers") {
    return Promise.resolve({ data: { computers: [computer("c1", "home"), computer("c2", "vps")] } });
  }
  if (url.endsWith("/projects")) {
    return Promise.reject(new Error("the computer is offline"));
  }
  return Promise.resolve({ data: {} });
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PairingDefaultsSection />
    </QueryClientProvider>,
  );
};

describe("PairingDefaultsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockImplementation(mockGet);
  });

  it("seeds the form from the loaded defaults", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/pairing/defaults") {
        return Promise.resolve({ data: { provider: "claude", model: "sonnet", placement: "spread", clone_automatically: true } });
      }
      return mockGet(url);
    });
    renderSection();

    expect(await screen.findByDisplayValue("claude")).toBeInTheDocument();
    expect(screen.getByDisplayValue("sonnet")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: /send runs to/i })).toHaveTextContent("Least busy");
  });

  it("saves updated defaults", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/pairing/defaults") return Promise.resolve({ data: { placement: "in_order", clone_automatically: false } });
      return mockGet(url);
    });
    mocks.put.mockResolvedValue({ data: { provider: "opencode", model: "", placement: "spread", clone_automatically: true, start_in: "worktree" } });
    const user = userEvent.setup();
    renderSection();

    await screen.findByRole("button", { name: /save defaults/i });
    await pickOption(user, /send runs to/i, "Least busy");
    await user.type(screen.getByLabelText(/^provider$/i), "opencode");
    await pickOption(user, /new threads start in/i, "New worktree per thread");
    await user.click(screen.getByRole("button", { name: /save defaults/i }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/pairing/defaults", {
        provider: "opencode",
        model: "",
        model_options: [],
        start_in: "worktree",
        placement: "spread",
        clone_automatically: false,
      }),
    );
  });
});
