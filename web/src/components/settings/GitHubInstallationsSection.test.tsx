import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GitHubInstallationsSection } from "@/components/settings/GitHubInstallationsSection";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: vi.fn() }));

const app = { configured: true, client_id: "Iv1.abc123", base_url: "", app_slug: "nexul-otal" };
const unregisteredApp = { configured: false };

const githubConnector = (configured: boolean) => [
  {
    connector: { id: "github", name: "GitHub", description: "", category: "source", icon: "github" },
    status: { configured },
    available: true,
    app_configured: true,
  },
];

const installations = [
  {
    id: 1,
    account_login: "otal-labs",
    account_type: "organization",
    account_avatar_url: "",
    repository_selection: "all",
    html_url: "https://github.com/organizations/otal-labs/settings/installations/1",
  },
  {
    id: 2,
    account_login: "onik",
    account_type: "user",
    account_avatar_url: "",
    repository_selection: "selected",
    repository_count: 2,
    html_url: "https://github.com/settings/installations/2",
  },
];

const stubApi = (appConfig: object, connected: boolean, accounts: object[] = installations) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/connectors/github/app-config") return { data: appConfig };
    if (url === "/api/connectors") return { data: githubConnector(connected) };
    if (url === "/api/repositories/installations") return { data: { installations: accounts } };
    throw new Error(`unexpected GET ${url}`);
  });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/settings/connectors/github-app"]}>
        <Routes>
          <Route path="/settings/connectors/:tab?" element={<GitHubInstallationsSection />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const addLink = () => screen.findByRole("link", { name: /add account or organisation/i });

describe("GitHubInstallationsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("lists each installation with its access and a Manage link when GitHub is connected", async () => {
    stubApi(app, true);
    renderSection();

    expect(await screen.findByText("otal-labs")).toBeInTheDocument();
    expect(screen.getByText("All repositories")).toBeInTheDocument();
    expect(screen.getByText("2 selected repositories")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage onik on GitHub" })).toHaveAttribute(
      "href",
      "https://github.com/settings/installations/2",
    );
    expect(await addLink()).toHaveAttribute("href", "https://github.com/apps/nexul-otal/installations/new");
  });

  it("says so when the App is installed nowhere the connector can see, still offering to add one", async () => {
    stubApi(app, true, []);
    renderSection();

    expect(await screen.findByText(/isn't installed on any account you can see yet/i)).toBeInTheDocument();
    expect(await addLink()).toBeInTheDocument();
  });

  it("points at the Connectors tab and keeps Add, without reading installations, when GitHub is not connected", async () => {
    stubApi(app, false);
    renderSection();

    expect(await screen.findByRole("link", { name: "Connectors tab" })).toHaveAttribute("href", "/settings/connectors");
    expect(screen.getByText(/to see where the App is installed/)).toBeInTheDocument();
    expect(await addLink()).toHaveAttribute("href", "https://github.com/apps/nexul-otal/installations/new");
    expect(mocks.get).not.toHaveBeenCalledWith("/api/repositories/installations");
  });

  it("renders nothing until the App is registered", async () => {
    stubApi(unregisteredApp, true);
    renderSection();

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/connectors/github/app-config"));
    expect(screen.queryByRole("heading", { name: "Installations" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /add account or organisation/i })).not.toBeInTheDocument();
  });
});
