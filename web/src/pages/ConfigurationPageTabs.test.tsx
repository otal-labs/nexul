import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConfigurationPage } from "@/pages/ConfigurationPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));
vi.mock("@/components/settings/RoleSettingsSection", () => ({ RoleSettingsSection: () => <p>Roles card</p> }));
vi.mock("@/components/team/TeamSection", () => ({ TeamSection: () => <p>Team card</p> }));

// Every instance-level permission, as the Owner of some workspace holds them.
const everythingAnywhere = ["instance:read", "accounts:read", "members:write", "connectors:read", "connectors:write", "dns:read"];

const routeGet = (anywhere: string[], permissions: string[]) => (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: {}, instance_permissions: anywhere } });
  if (url === "/api/workspaces/ws-1/me") return Promise.resolve({ data: { role_name: "Owner", permissions } });
  return Promise.reject(new Error(`unexpected GET ${url}`));
};

const LocationProbe = () => {
  const location = useLocation();
  return <output aria-label="location">{`${location.pathname}${location.search}${location.hash}`}</output>;
};

const renderPage = (route: string, anywhere = everythingAnywhere, permissions: string[] = []) => {
  mocks.get.mockImplementation(routeGet(anywhere, permissions));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/configuration/:section?" element={<ConfigurationPage />} />
          <Route path="/settings/:section?" element={<p>Settings page</p>} />
        </Routes>
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("ConfigurationPage sections", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("lists only workspace sections, even for a viewer holding every instance permission", async () => {
    renderPage("/configuration", everythingAnywhere, ["roles:write"]);

    const nav = within(await screen.findByRole("navigation", { name: "Configuration sections" }));
    expect(await screen.findByText("Roles card")).toBeInTheDocument();
    expect(nav.getAllByRole("link").map((link) => link.textContent)).toEqual(["Roles", "Danger zone"]);
  });

  it("lands an instance-only viewer on Danger zone, never on an instance section", async () => {
    renderPage("/configuration/automation-secrets", ["connectors:read"]);

    expect(await screen.findByRole("link", { name: "Danger zone" })).toHaveAttribute("aria-current", "page");
    expect(screen.queryByRole("link", { name: "Connectors" })).not.toBeInTheDocument();
  });

  it("keeps Team among the workspace sections of a members:write holder without accounts:read", async () => {
    renderPage("/configuration/team", ["members:write"], ["members:write"]);

    expect(await screen.findByText("Team card")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent("/configuration/team");
  });

  // The path-to-path mapping is SettingsRedirects' own table; these prove the page applies it and keeps query and hash.
  it.each([
    ["/configuration/instance#instance-version", "/settings/instance#instance-version"],
    ["/configuration/connectors?tab=github-app", "/settings/connectors?tab=github-app"],
    ["/configuration/access?person=u-1", "/settings/team?person=u-1"],
    ["/configuration/team?person=u-1", "/settings/team?person=u-1"],
  ])("sends the old link %s to %s", async (from, to) => {
    renderPage(from);

    expect(await screen.findByText("Settings page")).toBeInTheDocument();
    expect(screen.getByLabelText("location").textContent).toBe(to);
  });
});
