import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SettingsPage } from "@/pages/SettingsPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), toastSuccess: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: vi.fn() } }));

// Stub every card so these tests only exercise which tab a URL lands on; ConnectorsSection stays real to prove it still reads the OAuth callback.
vi.mock("@/components/settings/InstanceVersionSection", () => ({ InstanceVersionSection: () => <p>Version card</p> }));
vi.mock("@/components/settings/InstanceUrlSection", () => ({ InstanceUrlSection: () => <p>URL card</p> }));
vi.mock("@/components/settings/OAuthProviderSection", () => ({
  OAuthProviderSection: ({ provider }: { provider: string }) => <p>{provider} card</p>,
}));
vi.mock("@/components/settings/ConnectionTokenSection", () => ({ ConnectionTokenSection: () => <p>Connection card</p> }));
vi.mock("@/components/settings/PersonalAccessTokensSection", () => ({
  PersonalAccessTokensSection: () => <p>PAT card</p>,
}));
vi.mock("@/components/settings/ComputersSection", () => ({ ComputersSection: () => <p>Computers card</p> }));
vi.mock("@/components/settings/PairingDefaultsSection", () => ({ PairingDefaultsSection: () => <p>Defaults card</p> }));
vi.mock("@/components/settings/ConnectorAppConfigSection", () => ({ ConnectorAppConfigSection: () => <p>App card</p> }));

const settings = { instance_url: "https://deploy.example.com", settings_version: 1 };

const routeGet = (admin: boolean) => (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { can_create_workspace: admin } } });
  if (url === "/api/connectors") {
    return Promise.resolve({
      data: [{ connector: { id: "github", name: "GitHub" }, status: { configured: true } }],
    });
  }
  return Promise.resolve({ data: settings });
};

const LocationProbe = () => {
  const location = useLocation();
  return <output aria-label="location">{`${location.search}${location.hash}`}</output>;
};

const renderPage = (route: string, admin = true) => {
  mocks.get.mockImplementation(routeGet(admin));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <SettingsPage />
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const selectedTab = (name: string) => screen.findByRole("tab", { name, selected: true });

describe("SettingsPage tabs", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.toastSuccess.mockReset();
  });

  it("no longer lists automation secrets, and an old secrets link falls back to Instance", async () => {
    renderPage("/settings?section=automation-secrets");

    expect(await screen.findByText("URL card")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Automation secrets" })).not.toBeInTheDocument();
  });

  it("gives a non-admin only the General instance card, with no tab row", async () => {
    renderPage("/settings?section=instance&tab=sign-in", false);

    expect(await screen.findByText("URL card")).toBeInTheDocument();
    expect(screen.queryByText("google card")).not.toBeInTheDocument();
    expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
  });

  it("opens Instance on General, the version anchor's tab, and moves sign-in providers to their own tab", async () => {
    const user = userEvent.setup();
    renderPage("/settings?section=instance#instance-version");

    await selectedTab("General");
    expect(await screen.findByText("Version card")).toBeInTheDocument();
    expect(screen.getByText("URL card")).toBeInTheDocument();
    expect(screen.queryByText("google card")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Sign-in providers" }));
    expect(screen.getByText("google card")).toBeInTheDocument();
    expect(screen.getByText("discord card")).toBeInTheDocument();
    expect(screen.queryByText("URL card")).not.toBeInTheDocument();
  });

  it("splits tokens into Connection token and Personal tokens", async () => {
    const user = userEvent.setup();
    renderPage("/settings?section=tokens");

    await selectedTab("Connection token");
    expect(screen.getByText("Connection card")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Personal tokens" }));
    expect(screen.getByText("PAT card")).toBeInTheDocument();
    expect(screen.queryByText("Connection card")).not.toBeInTheDocument();
  });

  it("lands a computer setup link on the Computers tab with its setup param intact", async () => {
    const user = userEvent.setup();
    renderPage("/settings?section=pairing&setup=c1");

    await selectedTab("Computers");
    expect(screen.getByText("Computers card")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent("setup=c1");

    await user.click(screen.getByRole("tab", { name: "Defaults" }));
    expect(screen.getByText("Defaults card")).toBeInTheDocument();
  });

  it("lands the connector OAuth callback on the Connectors tab, which toasts and strips it", async () => {
    renderPage("/settings?section=connectors&connector=github&connected=1");

    await selectedTab("Connectors");
    await vi.waitFor(() => expect(mocks.toastSuccess).toHaveBeenCalledWith(expect.stringMatching(/connected$/)));
    await vi.waitFor(() => expect(screen.getByLabelText("location")).toHaveTextContent(/^\?section=connectors$/));
    expect(screen.queryByText("App card")).not.toBeInTheDocument();
  });

  it("keeps the GitHub App card on its own tab for admins and hides it from everyone else", async () => {
    const user = userEvent.setup();
    renderPage("/settings?section=connectors");

    await user.click(await screen.findByRole("tab", { name: "GitHub App" }));
    expect(screen.getByText("App card")).toBeInTheDocument();
  });

  it("drops the GitHub App tab for a non-admin", async () => {
    renderPage("/settings?section=connectors&tab=github-app", false);

    await screen.findByRole("heading", { name: "Settings" });
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/connectors"));
    expect(screen.queryByRole("tab", { name: "GitHub App" })).not.toBeInTheDocument();
    expect(screen.queryByText("App card")).not.toBeInTheDocument();
  });
});
