import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConfigurationPage } from "@/pages/ConfigurationPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "error" }));
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
          <Route path="/acme/configuration/:section?" element={<ConfigurationPage />} />
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
    renderPage("/acme/configuration", everythingAnywhere, ["workspaces:write", "roles:write", "workspaces:delete"]);

    const nav = within(await screen.findByRole("navigation", { name: "Configuration sections" }));
    await screen.findByRole("link", { name: "General" });
    expect(nav.getAllByRole("link").map((link) => link.textContent)).toEqual(["General", "Mention chips", "Danger zone"]);
  });

  it("leaves Danger zone out for a viewer whose role hides Archive, and sends a link to it to the first section", async () => {
    renderPage("/acme/configuration/danger", everythingAnywhere, ["workspaces:write"]);

    expect(await screen.findByRole("link", { name: "General" })).toHaveAttribute("aria-current", "page");
    expect(screen.queryByRole("link", { name: "Danger zone" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "General" })).toBeInTheDocument();
  });

  it("never opens an instance section for an instance-only viewer", async () => {
    renderPage("/acme/configuration/connectors", ["connectors:read"]);

    await screen.findByRole("navigation", { name: "Configuration sections" });
    expect(screen.queryByRole("link", { name: "Connectors" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { level: 1, name: "Connectors" })).not.toBeInTheDocument();
  });

  it("keeps Team among the workspace sections of a members:write holder without accounts:read", async () => {
    renderPage("/acme/configuration/team", ["members:write"], ["members:write"]);

    expect(await screen.findByText("Team card")).toBeInTheDocument();
    expect(screen.getByLabelText("location")).toHaveTextContent("/acme/configuration/team");
  });

  it("sends an accounts:read holder's Team to the instance-wide one on Settings, keeping the query", async () => {
    renderPage("/acme/configuration/team?person=u-1");

    expect(await screen.findByText("Settings page")).toBeInTheDocument();
    expect(screen.getByLabelText("location").textContent).toBe("/settings/team?person=u-1");
  });
});
