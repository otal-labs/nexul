import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectorAppConfigSection } from "@/components/settings/ConnectorAppConfigSection";

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

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ConnectorAppConfigSection />
    </QueryClientProvider>,
  );
};

describe("ConnectorAppConfigSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockReset();
  });

  it("renders the client id, client secret, base url, and app slug fields", () => {
    renderSection();
    expect(screen.getByLabelText(/client id/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/client secret/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/base url/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/app slug/i)).toBeInTheDocument();
  });

  it("renders the client secret as a password field", () => {
    renderSection();
    expect(screen.getByLabelText(/client secret/i)).toHaveAttribute("type", "password");
  });

  it("submits the entered values to the github app-config endpoint", async () => {
    mocks.put.mockResolvedValue({
      data: { configured: true, client_id: "Iv1.abc123", base_url: "", app_slug: "nexul-test-app" },
    });
    const user = userEvent.setup();
    renderSection();

    await user.type(screen.getByLabelText(/client id/i), "Iv1.abc123");
    await user.type(screen.getByLabelText(/client secret/i), "super-secret-value");
    await user.type(screen.getByLabelText(/app slug/i), "nexul-test-app");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/connectors/github/app-config", {
      client_id: "Iv1.abc123",
      client_secret: "super-secret-value",
      base_url: "",
      app_slug: "nexul-test-app",
    });
  });

  it("does not submit when client id or client secret is empty", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("submits the optional base url when provided", async () => {
    mocks.put.mockResolvedValue({
      data: {
        configured: true,
        client_id: "Iv1.abc123",
        base_url: "https://ghe.example.com",
        app_slug: "nexul-test-app",
      },
    });
    const user = userEvent.setup();
    renderSection();

    await user.type(screen.getByLabelText(/client id/i), "Iv1.abc123");
    await user.type(screen.getByLabelText(/client secret/i), "super-secret-value");
    await user.type(screen.getByLabelText(/base url/i), "https://ghe.example.com");
    await user.type(screen.getByLabelText(/app slug/i), "nexul-test-app");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/connectors/github/app-config", {
      client_id: "Iv1.abc123",
      client_secret: "super-secret-value",
      base_url: "https://ghe.example.com",
      app_slug: "nexul-test-app",
    });
  });

  it("surfaces a backend error inline", async () => {
    mocks.put.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Owner permission required");
    const user = userEvent.setup();
    renderSection();

    await user.type(screen.getByLabelText(/client id/i), "Iv1.abc123");
    await user.type(screen.getByLabelText(/client secret/i), "super-secret-value");
    await user.type(screen.getByLabelText(/app slug/i), "nexul-test-app");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(await screen.findByText("Owner permission required")).toBeInTheDocument();
  });
});
