import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const githubConnector = [
  {
    connector: { id: "github", name: "GitHub", description: "", category: "source", icon: "github" },
    status: { configured: false },
    available: true,
    app_configured: true,
  },
];

const route = (anywhere: string[], app: { configured: boolean }) => (url: string) => {
  if (url === "/api/auth/me") return Promise.resolve({ data: { user: {}, instance_permissions: anywhere } });
  if (url === "/api/connectors/github/app-config") return Promise.resolve({ data: { ...app, client_id: "Iv1.abc", base_url: "", app_slug: "nexul-app" } });
  if (url === "/api/connectors") return Promise.resolve({ data: githubConnector });
  return Promise.resolve({ data: [] });
};

const renderPanel = (anywhere: string[], app = { configured: true }) => {
  mocks.get.mockImplementation(route(anywhere, app));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/configuration/connectors?tab=github-app"]}>
        <ConnectorsSettingsPanel />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("ConnectorsSettingsPanel GitHub App tab", () => {
  beforeEach(() => mocks.get.mockReset());

  it("shows the App's management card to a connectors:write holder, beside Installations", async () => {
    renderPanel(["connectors:read", "connectors:write"]);

    expect(await screen.findByRole("heading", { name: "GitHub App" })).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Installations" })).toBeInTheDocument();
  });

  it("shows only Installations to a viewer without connectors:write", async () => {
    renderPanel(["connectors:read"]);

    expect(await screen.findByRole("heading", { name: "Installations" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "GitHub App" })).not.toBeInTheDocument();
  });

  it("drops the tab for a viewer who could see neither card", async () => {
    renderPanel(["connectors:read"], { configured: false });

    expect(await screen.findByRole("heading", { name: "Connectors" })).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "GitHub App" })).not.toBeInTheDocument();
  });
});
