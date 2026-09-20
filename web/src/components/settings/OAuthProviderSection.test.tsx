import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import type { OptionalProvider } from "@/models/User";

const mocks = vi.hoisted(() => ({
  put: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { put: mocks.put },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const settings = {
  instance_url: "https://deploy.example.com",
  settings_version: 2,
  oauth_callback: "https://deploy.example.com/auth/callback",
  mention_chip_template: "",
  google_oauth_client_id: "",
  google_oauth_callback: "https://deploy.example.com/auth/google/callback",
  google_oauth_configured: false,
  discord_oauth_client_id: "",
  discord_oauth_callback: "https://deploy.example.com/auth/discord/callback",
  discord_oauth_configured: false,
};

const renderSection = (provider: OptionalProvider, overrides: Partial<typeof settings> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <OAuthProviderSection provider={provider} settings={{ ...settings, ...overrides }} />
    </QueryClientProvider>,
  );
};

describe("OAuthProviderSection", () => {
  beforeEach(() => {
    mocks.put.mockReset();
    mocks.put.mockResolvedValue({ data: { configured: true } });
  });

  it("shows the provider's own redirect uri to register", () => {
    renderSection("discord");
    expect(screen.getByText("https://deploy.example.com/auth/discord/callback")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /discord sign-in/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /disable discord sign-in/i })).not.toBeInTheDocument();
  });

  it("saves client id + secret to the provider's route", async () => {
    const user = userEvent.setup();
    renderSection("google");
    await user.type(screen.getByLabelText(/client id/i), "g-id");
    await user.type(screen.getByLabelText(/client secret/i), "g-secret");
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings/oauth/google", { client_id: "g-id", client_secret: "g-secret" });
  });

  it("rejects a half-filled form without calling the api", async () => {
    const user = userEvent.setup();
    renderSection("discord");
    await user.type(screen.getByLabelText(/client id/i), "d-id");
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    expect(mocks.put).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent(/both/i);
  });

  it("disables by clearing both when already configured", async () => {
    const user = userEvent.setup();
    renderSection("discord", { discord_oauth_client_id: "d-id", discord_oauth_configured: true });
    expect(screen.getByLabelText(/client id/i)).toHaveValue("d-id");
    await user.click(screen.getByRole("button", { name: /disable discord sign-in/i }));
    expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings/oauth/discord", { client_id: "", client_secret: "" });
  });
});
