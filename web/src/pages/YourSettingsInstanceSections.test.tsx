import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { YourSettingsPage } from "@/pages/YourSettingsPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), toastSuccess: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "Settings failed" }));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: vi.fn() } }));

// Personal sections are covered by YourSettingsPage.test; stub them so these tests only exercise the instance group.
vi.mock("@/components/you/YourSettingsContent", () => ({
  YourSettingsContent: ({ section }: { section: string }) => <p>{section} card</p>,
}));
// The other instance cards are stubbed too; ConnectorsSection stays real to prove it still reads the OAuth callback.
vi.mock("@/components/settings/InstanceVersionSection", () => ({ InstanceVersionSection: () => <p>Version card</p> }));
vi.mock("@/components/settings/InstanceUrlSection", () => ({ InstanceUrlSection: () => <p>URL card</p> }));
vi.mock("@/components/settings/OAuthProviderSection", () => ({
  OAuthProviderSection: ({ provider }: { provider: string }) => <p>{provider} card</p>,
}));
vi.mock("@/components/settings/ConnectorAppConfigSection", () => ({ ConnectorAppConfigSection: () => <p>App card</p> }));
vi.mock("@/components/team/TeamSection", () => ({ TeamSection: () => <p>Team card</p> }));

const settings = { instance_url: "https://deploy.example.com", settings_version: 1 };

// The Owner of a workspace holds every instance bit.
const ownerBits = ["instance:read", "accounts:read", "members:write", "connectors:read", "connectors:write", "dns:read", "templates:write"];

const routeGet = (anywhere: string[], permissions: string[] = []) => (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: {}, instance_permissions: anywhere } });
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Member", permissions } });
  if (url === "/api/team") return Promise.reject(new Error("403"));
  if (url === "/api/connectors") {
    return Promise.resolve({ data: [{ connector: { id: "github", name: "GitHub" }, status: { configured: true } }] });
  }
  return Promise.resolve({ data: settings });
};

const LocationProbe = () => {
  const location = useLocation();
  return <output aria-label="location">{`${location.pathname}${location.search}${location.hash}`}</output>;
};

const renderPage = (route: string, anywhere = ownerBits, permissions: string[] = []) => {
  mocks.get.mockImplementation(routeGet(anywhere, permissions));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/settings/:section?/:tab?" element={<YourSettingsPage />} />
          <Route path="/acme/configuration/:section?" element={<p>Configuration page</p>} />
        </Routes>
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const selectedTab = (name: string) => screen.findByRole("tab", { name, selected: true });

describe("Settings page instance sections", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.toastSuccess.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  // Each section opens with its own permission held in any workspace, which /me reports.
  it.each([
    { held: ownerBits, shown: ["Instance", "Team", "Sign-in providers", "Connectors", "DNS", "Templates"] },
    { held: ["instance:read"], shown: ["Instance", "Sign-in providers"] },
    // Every member reads the instance templates through their workspaces; only templates:write opens the editor.
    { held: ["instance:read", "templates:read"], shown: ["Instance", "Sign-in providers"] },
    { held: ["templates:write"], shown: ["Templates"] },
    { held: ["connectors:read"], shown: ["Connectors"] },
    { held: ["dns:read"], shown: ["DNS"] },
    { held: ["accounts:read"], shown: ["Team"] },
  ])("with $held the Instance settings group lists $shown", async ({ held, shown }) => {
    renderPage("/settings", held);

    const nav = within(await screen.findByRole("navigation", { name: "Settings sections" }));
    await nav.findByText("Instance settings");
    expect(nav.getByText("You")).toBeInTheDocument();
    const instanceLinks = ["Instance", "Team", "Sign-in providers", "Connectors", "DNS", "Templates"].filter((name) => nav.queryByRole("link", { name }));
    expect(instanceLinks).toEqual(shown);
  });

  it("gives a plain member no Instance settings group and no label on their own list", async () => {
    renderPage("/settings", []);

    const nav = within(await screen.findByRole("navigation", { name: "Settings sections" }));
    expect(await screen.findByText("profile card")).toBeInTheDocument();
    expect(nav.getAllByRole("link").map((link) => link.textContent)).toEqual(["Profile", "Appearance", "Security", "Computers"]);
    expect(nav.queryByText("Instance settings")).not.toBeInTheDocument();
    expect(nav.queryByText("You")).not.toBeInTheDocument();
  });

  it("links an instance section under /settings", async () => {
    renderPage("/settings");

    const nav = within(await screen.findByRole("navigation", { name: "Settings sections" }));
    expect(await nav.findByRole("link", { name: "Sign-in providers" })).toHaveAttribute("href", "/settings/sign-in");
    expect(nav.getByRole("link", { name: "Team" })).toHaveAttribute("href", "/settings/team");
  });

  it("falls back to Profile when the link names an instance section the viewer can't open", async () => {
    renderPage("/settings/instance", []);

    expect(await screen.findByText("profile card")).toBeInTheDocument();
    expect(screen.queryByText("URL card")).not.toBeInTheDocument();
  });

  it("shows an error, not a blank page, when the instance settings fail to load", async () => {
    mocks.get.mockImplementation((url: string) =>
      url === "/api/auth/settings" ? Promise.reject(new Error("boom")) : routeGet(ownerBits)(url),
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={["/settings/instance"]}>
          <Routes>
            <Route path="/settings/:section?/:tab?" element={<YourSettingsPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(await screen.findByText("Settings failed")).toBeInTheDocument();
  });

  it("sends a members:write holder without accounts:read from Team to the scoped one in Configuration", async () => {
    renderPage("/settings/team?person=u-1", ["members:write"]);

    expect(await screen.findByText("Configuration page")).toBeInTheDocument();
    expect(screen.getByLabelText("location").textContent).toBe("/acme/configuration/team?person=u-1");
  });

  it("gives sign-in providers their own section, one tab each with Discord first", async () => {
    renderPage("/settings/sign-in");

    await selectedTab("Discord");
    expect(screen.getByText("discord card")).toBeInTheDocument();
    expect(screen.queryByText("URL card")).not.toBeInTheDocument();
  });

  it("opens the Google sign-in tab from its path", async () => {
    renderPage("/settings/sign-in/google");

    await selectedTab("Google");
    expect(screen.getByText("google card")).toBeInTheDocument();
  });

  it("lands the connector OAuth callback on the Connectors tab, which toasts and strips it", async () => {
    renderPage("/settings/connectors?connector=github&connected=1");

    await selectedTab("Connectors");
    await vi.waitFor(() => expect(mocks.toastSuccess).toHaveBeenCalledWith(expect.stringMatching(/connected$/)));
    await vi.waitFor(() => expect(screen.getByLabelText("location")).toHaveTextContent(/^\/settings\/connectors$/));
    expect(screen.queryByText("App card")).not.toBeInTheDocument();
  });

  it("keeps the GitHub App card on its own tab", async () => {
    const user = userEvent.setup();
    renderPage("/settings/connectors");

    await user.click(await screen.findByRole("tab", { name: "GitHub App" }));
    expect(screen.getByText("App card")).toBeInTheDocument();
  });
});
