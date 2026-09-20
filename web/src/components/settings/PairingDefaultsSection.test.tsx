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
  updated_at: "2026-08-01T00:00:00Z",
});

const mockGet = (url: string) => {
  if (url === "/api/pairing/computers") {
    return Promise.resolve({ data: { computers: [computer("c1", "home"), computer("c2", "vps")] } });
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
        return Promise.resolve({
          data: { default_computer_id: "c1", fallback_project_id: "proj-1", provider: "claude", model: "sonnet" },
        });
      }
      return mockGet(url);
    });
    renderSection();

    expect(await screen.findByDisplayValue("proj-1")).toBeInTheDocument();
    expect(screen.getByDisplayValue("claude")).toBeInTheDocument();
    expect(screen.getByDisplayValue("sonnet")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: /default computer/i })).toHaveTextContent("home");
  });

  it("saves updated defaults", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/pairing/defaults") return Promise.resolve({ data: {} });
      return mockGet(url);
    });
    mocks.put.mockResolvedValue({
      data: { default_computer_id: "c2", fallback_project_id: "proj-9", provider: "opencode", model: "" },
    });
    const user = userEvent.setup();
    renderSection();

    await screen.findByRole("button", { name: /save defaults/i });
    await pickOption(user, /default computer/i, "vps");
    await user.type(screen.getByLabelText(/fallback t3 project/i), "proj-9");
    await user.type(screen.getByLabelText(/^provider$/i), "opencode");
    await user.click(screen.getByRole("button", { name: /save defaults/i }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/api/pairing/defaults", {
        default_computer_id: "c2",
        fallback_project_id: "proj-9",
        provider: "opencode",
        model: "",
      }),
    );
  });
});
