import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConfigurationPage } from "@/pages/ConfigurationPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), toastSuccess: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: vi.fn() } }));

// Stub every card so these tests only exercise which section and tab a URL lands on; ConnectorsSection stays real to prove it still reads the OAuth callback.
vi.mock("@/components/settings/InstanceVersionSection", () => ({ InstanceVersionSection: () => <p>Version card</p> }));
vi.mock("@/components/settings/InstanceUrlSection", () => ({ InstanceUrlSection: () => <p>URL card</p> }));
vi.mock("@/components/settings/OAuthProviderSection", () => ({
  OAuthProviderSection: ({ provider }: { provider: string }) => <p>{provider} card</p>,
}));
vi.mock("@/components/settings/ConnectorAppConfigSection", () => ({ ConnectorAppConfigSection: () => <p>App card</p> }));
vi.mock("@/components/settings/RoleSettingsSection", () => ({ RoleSettingsSection: () => <p>Roles card</p> }));
vi.mock("@/components/team/TeamSection", () => ({ TeamSection: () => <p>Team card</p> }));

const settings = { instance_url: "https://deploy.example.com", settings_version: 1 };

const routeGet = (admin: boolean, permissions: string[]) => (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: { can_create_workspace: admin } } });
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Owner", permissions } });
  if (url === "/api/team") return Promise.reject(new Error("403"));
  if (url === "/api/connectors") {
    return Promise.resolve({
      data: [{ connector: { id: "github", name: "GitHub" }, status: { configured: true } }],
    });
  }
  return Promise.resolve({ data: settings });
};

const LocationProbe = () => {
  const location = useLocation();
  return <output aria-label="location">{`${location.pathname}${location.search}${location.hash}`}</output>;
};

const renderPage = (route: string, admin = true, permissions: string[] = []) => {
  mocks.get.mockImplementation(routeGet(admin, permissions));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/configuration/:section?" element={<ConfigurationPage />} />
        </Routes>
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const selectedTab = (name: string) => screen.findByRole("tab", { name, selected: true });

describe("ConfigurationPage sections", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.toastSuccess.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("lands an admin with no workspace permissions on Instance, never on Danger zone", async () => {
    renderPage("/configuration/automation-secrets");

    expect(await screen.findByText("URL card")).toBeInTheDocument();
    expect(screen.getByText("Version card")).toBeInTheDocument();
    expect(screen.queryByRole("tablist")).not.toBeInTheDocument();
  });

  it("opens the first workspace section for a workspace owner", async () => {
    renderPage("/configuration", true, ["roles:write", "members:write"]);

    expect(await screen.findByText("Roles card")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Team" })).toHaveAttribute("href", "/configuration/team");
  });

  it.each(["/configuration/members", "/configuration/access?person=u-1"])("lands the folded section link %s on Team", async (route) => {
    renderPage(route);

    expect(await screen.findByText("Team card")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent(/^\/configuration\/team(\?person=u-1)?$/);
  });

  it("shows Team among the workspace sections to a members:write holder who isn't an instance admin", async () => {
    renderPage("/configuration/team", false, ["members:write"]);

    expect(await screen.findByText("Team card")).toBeInTheDocument();
    expect(screen.queryByText("Whole instance")).not.toBeInTheDocument();
  });

  it("hides every whole-instance section from a non-admin, and an instance link falls back", async () => {
    renderPage("/configuration/instance", false);

    await screen.findByRole("heading", { name: "Configuration" });
    expect(screen.queryByText("URL card")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Instance" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Connectors" })).not.toBeInTheDocument();
    expect(screen.queryByText("Whole instance")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Danger zone" })).toHaveAttribute("aria-current", "page");
  });

  it("gives sign-in providers their own section, one tab each with Discord first", async () => {
    renderPage("/configuration/sign-in");

    await selectedTab("Discord");
    expect(screen.getByText("discord card")).toBeInTheDocument();
    expect(screen.queryByText("URL card")).not.toBeInTheDocument();
  });

  it("opens the Google sign-in tab from its ?tab=", async () => {
    renderPage("/configuration/sign-in?tab=google");

    await selectedTab("Google");
    expect(screen.getByText("google card")).toBeInTheDocument();
  });

  it("lands the connector OAuth callback on the Connectors tab, which toasts and strips it", async () => {
    renderPage("/configuration/connectors?connector=github&connected=1");

    await selectedTab("Connectors");
    await vi.waitFor(() => expect(mocks.toastSuccess).toHaveBeenCalledWith(expect.stringMatching(/connected$/)));
    await vi.waitFor(() => expect(screen.getByLabelText("location")).toHaveTextContent(/^\/configuration\/connectors$/));
    expect(screen.queryByText("App card")).not.toBeInTheDocument();
  });

  it("keeps the GitHub App card on its own tab", async () => {
    const user = userEvent.setup();
    renderPage("/configuration/connectors");

    await user.click(await screen.findByRole("tab", { name: "GitHub App" }));
    expect(screen.getByText("App card")).toBeInTheDocument();
  });
});
