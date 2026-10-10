import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GitHubInstallationsSection } from "@/components/settings/GitHubInstallationsSection";
import { ConnectorAppConfigSection } from "@/components/settings/ConnectorAppConfigSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ConnectorAppConfigSection />
    </QueryClientProvider>,
  );
};

// An instance whose GitHub App was never registered shows the registration form.
const renderUnregistered = async () => {
  renderSection();
  await screen.findByLabelText(/client id/i);
};

const registered = { configured: true, client_id: "Iv1.abc123", base_url: "", app_slug: "nexul-otal" };

describe("ConnectorAppConfigSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockReset();
    mocks.get.mockResolvedValue({ data: { configured: false } });
  });

  it("renders the client id, client secret, base url, and app slug fields", async () => {
    await renderUnregistered();
    expect(screen.getByLabelText(/client id/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/client secret/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/base url/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/app slug/i)).toBeInTheDocument();
  });

  it("renders the client secret as a password field", async () => {
    await renderUnregistered();
    expect(screen.getByLabelText(/client secret/i)).toHaveAttribute("type", "password");
  });

  it("submits the entered values to the github app-config endpoint", async () => {
    mocks.put.mockResolvedValue({
      data: { configured: true, client_id: "Iv1.abc123", base_url: "", app_slug: "nexul-test-app" },
    });
    const user = userEvent.setup();
    await renderUnregistered();

    await user.type(screen.getByLabelText(/client id/i), "Iv1.abc123");
    await user.type(screen.getByLabelText(/client secret/i), "super-secret-value");
    await user.type(screen.getByLabelText(/app slug/i), "nexul-test-app");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/connectors/github/app-config", {
      client_id: "Iv1.abc123",
      client_secret: "super-secret-value",
      base_url: "",
      app_slug: "nexul-test-app",
    });
  });

  it("does not submit when client id or client secret is empty", async () => {
    const user = userEvent.setup();
    await renderUnregistered();

    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("submits the optional base url when provided", async () => {
    mocks.put.mockResolvedValue({
      data: {
        configured: true,
        client_id: "Iv1.abc123",
        base_url: "https://ghe.example.com",
        app_slug: "nexul-test-app",
      },
    });
    const user = userEvent.setup();
    await renderUnregistered();

    await user.type(screen.getByLabelText(/client id/i), "Iv1.abc123");
    await user.type(screen.getByLabelText(/client secret/i), "super-secret-value");
    await user.type(screen.getByLabelText(/base url/i), "https://ghe.example.com");
    await user.type(screen.getByLabelText(/app slug/i), "nexul-test-app");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(mocks.put).toHaveBeenCalledWith("/api/connectors/github/app-config", {
      client_id: "Iv1.abc123",
      client_secret: "super-secret-value",
      base_url: "https://ghe.example.com",
      app_slug: "nexul-test-app",
    });
  });

  it("surfaces a backend error inline", async () => {
    mocks.put.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Owner permission required");
    const user = userEvent.setup();
    await renderUnregistered();

    await user.type(screen.getByLabelText(/client id/i), "Iv1.abc123");
    await user.type(screen.getByLabelText(/client secret/i), "super-secret-value");
    await user.type(screen.getByLabelText(/app slug/i), "nexul-test-app");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    expect(await screen.findByText("Owner permission required")).toBeInTheDocument();
  });

  describe("once the App is registered", () => {
    beforeEach(() => {
      mocks.get.mockResolvedValue({ data: registered });
    });

    it("shows the App, its client ID, and the server instead of an empty form", async () => {
      renderSection();

      expect(await screen.findByRole("link", { name: /nexul-otal/ })).toHaveAttribute(
        "href",
        "https://github.com/apps/nexul-otal",
      );
      expect(screen.getByText("Iv1.abc123")).toBeInTheDocument();
      expect(screen.getByText("github.com")).toBeInTheDocument();
      expect(screen.queryByLabelText(/client secret/i)).not.toBeInTheDocument();
      expect(mocks.get).toHaveBeenCalledWith("/api/connectors/github/app-config");
    });

    it("links an Enterprise App on its own server", async () => {
      mocks.get.mockResolvedValue({ data: { ...registered, base_url: "https://ghe.example.com/" } });
      renderSection();

      expect(await screen.findByRole("link", { name: /nexul-otal/ })).toHaveAttribute(
        "href",
        "https://ghe.example.com/apps/nexul-otal",
      );
    });

    it("edits in a dialog prefilled with the App, keeping the secret when left blank", async () => {
      mocks.put.mockResolvedValue({ data: { ...registered, app_slug: "nexul-renamed" } });
      const user = userEvent.setup();
      renderSection();

      await user.click(await screen.findByRole("button", { name: /^edit$/i }));
      const dialog = await screen.findByRole("dialog");
      expect(within(dialog).getByLabelText(/client id/i)).toHaveValue("Iv1.abc123");
      expect(within(dialog).getByLabelText(/client secret/i)).toHaveValue("");

      await user.clear(within(dialog).getByLabelText(/app slug/i));
      await user.type(within(dialog).getByLabelText(/app slug/i), "nexul-renamed");
      await user.click(within(dialog).getByRole("button", { name: /^save$/i }));

      expect(mocks.put).toHaveBeenCalledWith("/api/connectors/github/app-config", {
        client_id: "Iv1.abc123",
        client_secret: "",
        base_url: "",
        app_slug: "nexul-renamed",
      });
      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    });

    it("signals the missing private key and stores one only after GitHub accepts it", async () => {
      const key = "-----BEGIN RSA PRIVATE KEY-----";
      mocks.post.mockResolvedValue({ data: undefined });
      mocks.put.mockResolvedValue({ data: { ...registered, private_key_set: true } });
      let asApp = false;
      const installation = (account: string) => ({ id: 1, account_login: account, account_type: "Organization", account_avatar_url: "", repository_selection: "all", html_url: "https://github.example.com/installations/1", workspaces: [] });
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/connectors/github/app-config") return { data: registered };
        if (url === "/api/connectors") return { data: [{ connector: { id: "github" }, status: { configured: true } }] };
        if (url === "/api/auth/me") return { data: { instance_permissions: [] } };
        if (url === "/api/repositories/installations") return { data: { installations: [installation(asApp ? "globex" : "acme")] } };
        throw new Error(`unexpected GET ${url}`);
      });
      mocks.put.mockImplementation(async () => {
        asApp = true;
        return { data: { ...registered, private_key_set: true } };
      });
      const user = userEvent.setup();
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(<QueryClientProvider client={client}><MemoryRouter><ConnectorAppConfigSection /><GitHubInstallationsSection /></MemoryRouter></QueryClientProvider>);
      expect(await screen.findByText("acme")).toBeInTheDocument();

      expect(await screen.findByText("only the connected account's repositories are visible", { exact: false })).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: "Add private key" }));
      const dialog = await screen.findByRole("dialog");
      await user.type(within(dialog).getByLabelText("Private key"), key);
      await user.click(within(dialog).getByRole("button", { name: "Verify" }));

      expect(mocks.post).toHaveBeenCalledWith("/api/connectors/github/private-key/verify", { private_key: key });
      expect(mocks.put).not.toHaveBeenCalled();
      await user.click(await within(dialog).findByRole("button", { name: "Save" }));
      expect(mocks.put).toHaveBeenCalledWith("/api/connectors/github/private-key", { private_key: key });
      expect(await screen.findByText("Nexul reads every installation as the App", { exact: false })).toBeInTheDocument();
      expect(await screen.findByText("globex")).toBeInTheDocument();
      expect(screen.queryByText("acme")).not.toBeInTheDocument();
    });
  });
});
