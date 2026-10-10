import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GitHubInstallationsSection } from "@/components/settings/GitHubInstallationsSection";

const mocks = vi.hoisted(() => ({ get: vi.fn(), delete: vi.fn(), confirm: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, delete: mocks.delete }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));

const app = { configured: true, client_id: "Iv1.abc123", base_url: "", app_slug: "nexul-acme" };
const unregisteredApp = { configured: false };

const installations = [
  {
    id: 1,
    account_id: 11,
    account_login: "acme",
    account_type: "organization",
    account_avatar_url: "",
    repository_selection: "all",
    html_url: "https://github.com/organizations/acme/settings/installations/1",
    workspaces: [
      { id: "ws-1", name: "Acme", can_detach: true },
      { id: "ws-2", name: "Globex", can_detach: false },
    ],
  },
  {
    id: 2,
    account_id: 12,
    account_login: "alice",
    account_type: "user",
    account_avatar_url: "",
    repository_selection: "selected",
    repository_count: 2,
    html_url: "https://github.com/settings/installations/2",
    workspaces: [],
  },
];

const stubApi = (appConfig: object, link: object, accounts: object[] = installations) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/connectors/github/app-config") return { data: appConfig };
    if (url === "/api/auth/github-link") return { data: link };
    if (url === "/api/repositories/installations") return { data: { installations: accounts } };
    throw new Error(`unexpected GET ${url}`);
  });

const connected = { state: "connected", login: "alice" };

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
    mocks.delete.mockReset();
    mocks.confirm.mockReset();
  });

  it("lists each installation the viewer's GitHub sees, with its access and a Manage link", async () => {
    stubApi(app, connected);
    renderSection();

    expect(await screen.findByText("acme")).toBeInTheDocument();
    expect(screen.getByText("All repositories")).toBeInTheDocument();
    expect(screen.getByText("2 selected repositories")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage alice on GitHub" })).toHaveAttribute("href", "https://github.com/settings/installations/2");
    expect(await addLink()).toHaveAttribute("href", "https://github.com/apps/nexul-acme/installations/new");
    expect(screen.getByText(/A workspace uses one once someone who can open one of its repositories attaches it/)).toBeInTheDocument();
  });

  it("shows the workspaces that use each account and no way to assign one by hand", async () => {
    stubApi(app, connected);
    renderSection();

    expect(await screen.findByText("Globex")).toBeInTheDocument();
    expect(screen.getAllByText("Used by")).toHaveLength(2);
    expect(screen.getByText("Not used by a workspace yet")).toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Detach acme from Globex" })).not.toBeInTheDocument();
  });

  it("detaches an account from a workspace the viewer manages only once they confirm", async () => {
    stubApi(app, connected);
    mocks.delete.mockResolvedValue({});
    renderSection();
    const user = userEvent.setup();
    const detach = await screen.findByRole("button", { name: "Detach acme from Acme" });

    mocks.confirm.mockResolvedValueOnce(false);
    await user.click(detach);
    expect(mocks.delete).not.toHaveBeenCalled();

    mocks.confirm.mockResolvedValueOnce(true);
    await user.click(detach);
    await vi.waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/repositories/installations/acme/workspaces/ws-1"));
  });

  it("shows a refused installation's problem beside the others", async () => {
    const refused = { ...installations[1], problem: "GitHub refused the App access to it; check the installation on GitHub" };
    stubApi(app, connected, [installations[0]!, refused]);
    renderSection();

    expect(await screen.findByText(/GitHub refused the App access to it/)).toBeInTheDocument();
    expect(screen.getByText("acme")).toBeInTheDocument();
  });

  it("says so when the App is installed nowhere the viewer's GitHub can open, still offering to add one", async () => {
    stubApi(app, connected, []);
    renderSection();

    expect(await screen.findByText(/isn't installed on any account your GitHub can open yet/i)).toBeInTheDocument();
    expect(await addLink()).toBeInTheDocument();
  });

  it("asks a viewer without a GitHub link to connect it, without reading installations, and keeps Add", async () => {
    stubApi(app, { state: "none" });
    renderSection();

    expect(await screen.findByText("Connect GitHub to see your repositories")).toBeInTheDocument();
    expect(await addLink()).toHaveAttribute("href", "https://github.com/apps/nexul-acme/installations/new");
    expect(mocks.get).not.toHaveBeenCalledWith("/api/repositories/installations");
  });

  it("renders nothing until the App is registered", async () => {
    stubApi(unregisteredApp, connected);
    renderSection();

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/connectors/github/app-config"));
    expect(screen.queryByRole("heading", { name: "Installations" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /add account or organisation/i })).not.toBeInTheDocument();
  });
});
