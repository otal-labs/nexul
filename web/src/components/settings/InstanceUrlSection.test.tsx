import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";

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

const settings = {
  instance_url: "https://deploy.example.com",
  settings_version: 2,
  oauth_callback: "https://deploy.example.com/auth/callback",
  mention_chip_template: "{ticket.Ticket} {ticket.Status}",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <InstanceUrlSection settings={settings} />
    </QueryClientProvider>,
  );
};

describe("InstanceUrlSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
  });

  it("seeds the form with the current instance url", () => {
    renderSection();
    expect(screen.getByLabelText(/instance url/i)).toHaveValue("https://deploy.example.com");
  });

  it("shows the derived oauth callback", () => {
    renderSection();
    expect(screen.getByText(/https:\/\/deploy\.example\.com\/auth\/callback/i)).toBeInTheDocument();
  });

  it("saves a new instance url", async () => {
    mocks.put.mockResolvedValue({
      data: { ...settings, instance_url: "https://new.example.com", settings_version: 3 },
    });
    const user = userEvent.setup();
    renderSection();

    const input = screen.getByLabelText(/instance url/i);
    await user.clear(input);
    await user.type(input, "https://new.example.com");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings", {
      instance_url: "https://new.example.com",
    });
  });

  it("does not call the api for an invalid url", async () => {
    const user = userEvent.setup();
    renderSection();

    const input = screen.getByLabelText(/instance url/i);
    await user.clear(input);
    await user.type(input, "not-a-url");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).not.toHaveBeenCalled();
    expect(screen.getByText(/enter a valid url/i)).toBeInTheDocument();
  });

  it("shows an error when the save fails and keeps the form open", async () => {
    mocks.put.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Save failed");
    const user = userEvent.setup();
    renderSection();

    const input = screen.getByLabelText(/instance url/i);
    await user.clear(input);
    await user.type(input, "https://new.example.com");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(screen.getByLabelText(/instance url/i)).toHaveValue("https://new.example.com");
  });
});
