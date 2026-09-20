import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectionTokenSection } from "@/components/settings/ConnectionTokenSection";

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

const token = {
  token: "header.payload.sig",
  instance_url: "https://deploy.example.com",
  settings_version: 2,
  expires_at: "2026-09-11T12:00:00Z",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ConnectionTokenSection />
    </QueryClientProvider>,
  );
};

describe("ConnectionTokenSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
  });

  it("keeps the token hidden until generation succeeds", () => {
    renderSection();
    expect(screen.queryByText("header.payload.sig")).not.toBeInTheDocument();
  });

  it("generates and reveals a connection token", async () => {
    mocks.post.mockResolvedValue({ data: token });
    const user = userEvent.setup();
    renderSection();

    await user.click(screen.getByRole("button", { name: /generate connection token/i }));

    expect(await screen.findByText("header.payload.sig")).toBeInTheDocument();
    expect(screen.getByText(/instance: https:\/\/deploy\.example\.com/i)).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/connection-token");
  });

  it("shows an error when generation fails and does not reveal a token", async () => {
    mocks.post.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Generation failed");
    const user = userEvent.setup();
    renderSection();

    await user.click(screen.getByRole("button", { name: /generate connection token/i }));

    expect(screen.queryByText("header.payload.sig")).not.toBeInTheDocument();
    expect(screen.queryByText(/copy this token now/i)).not.toBeInTheDocument();
  });

  it("hides a revealed token when the settings version changes", async () => {
    mocks.post.mockResolvedValue({ data: token });
    const user = userEvent.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { rerender } = render(
      <QueryClientProvider client={client}>
        <ConnectionTokenSection key={2} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: /generate connection token/i }));
    expect(await screen.findByText("header.payload.sig")).toBeInTheDocument();

    rerender(
      <QueryClientProvider client={client}>
        <ConnectionTokenSection key={3} />
      </QueryClientProvider>,
    );

    expect(screen.queryByText("header.payload.sig")).not.toBeInTheDocument();
  });
});
