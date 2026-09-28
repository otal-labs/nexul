import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
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
      <ContextAwareConfirmation.ConfirmationRoot />
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

  it("enables with client id + secret on the provider's route", async () => {
    const user = userEvent.setup();
    renderSection("google");
    await user.type(screen.getByLabelText(/client id/i), "g-id");
    await user.type(screen.getByLabelText(/client secret/i), "g-secret");
    await user.click(screen.getByRole("button", { name: "Enable Google sign-in" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings/oauth/google", { client_id: "g-id", client_secret: "g-secret" });
  });

  it("refuses a first setup without a secret, without calling the api", async () => {
    const user = userEvent.setup();
    renderSection("discord");
    await user.type(screen.getByLabelText(/client id/i), "d-id");
    await user.click(screen.getByRole("button", { name: "Enable Discord sign-in" }));
    expect(mocks.put).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent(/client secret is required/i);
  });

  describe("once enabled", () => {
    const enabled = { discord_oauth_client_id: "d-id", discord_oauth_configured: true };

    it("shows it is enabled with its client id and redirect uri, and no secret field", () => {
      renderSection("discord", enabled);
      expect(screen.getByText("Enabled")).toBeInTheDocument();
      expect(screen.getByText("d-id")).toBeInTheDocument();
      expect(screen.getAllByText("https://deploy.example.com/auth/discord/callback").length).toBeGreaterThan(0);
      expect(screen.queryByLabelText(/client secret/i)).not.toBeInTheDocument();
    });

    it("edits the client id in a dialog and keeps the stored secret when it is left blank", async () => {
      const user = userEvent.setup();
      renderSection("discord", enabled);
      await user.click(screen.getByRole("button", { name: "Edit" }));
      const dialog = await screen.findByRole("dialog");
      await user.clear(within(dialog).getByLabelText(/client id/i));
      await user.type(within(dialog).getByLabelText(/client id/i), "d-id-2");
      await user.click(within(dialog).getByRole("button", { name: "Save" }));
      expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings/oauth/discord", { client_id: "d-id-2", client_secret: "" });
    });

    it("asks before disabling, then clears both", async () => {
      const user = userEvent.setup();
      renderSection("discord", enabled);
      await user.click(screen.getByRole("button", { name: "Disable" }));
      expect(mocks.put).not.toHaveBeenCalled();
      const confirmation = await screen.findByRole("dialog");
      await user.click(within(confirmation).getByRole("button", { name: "Disable" }));
      expect(mocks.put).toHaveBeenCalledWith("/api/auth/settings/oauth/discord", { client_id: "", client_secret: "" });
    });
  });
});
