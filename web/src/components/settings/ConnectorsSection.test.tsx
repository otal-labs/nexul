import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectorsSection } from "@/components/settings/ConnectorsSection";
import type { ConnectorStatus } from "@/models/Connectors";

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

const toastMocks = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast: toastMocks }));

// Mirrors real backend state: github wired for OAuth, cloudflare "coming
// soon" (OAuth: nil), none connected yet.
const connectorEntry = (overrides: Partial<ConnectorStatus> = {}): ConnectorStatus => ({
  connector: {
    id: "github",
    name: "GitHub",
    description: "Links pull requests and reviews to tickets, scans your repositories, and deploys on push",
    category: "development",
    icon: "github",
  },
  status: { configured: false },
  available: true,
  app_configured: true,
  ...overrides,
});

const owner = { user: { can_create_workspace: true }, needs_owner_wizard: false, needs_first_login_wizard: false };
const member = { user: { can_create_workspace: false }, needs_owner_wizard: false, needs_first_login_wizard: false };

// Cloudflare wired for OAuth but with no app registration stored yet.
const unregisteredCloudflare = (): ConnectorStatus =>
  connectorEntry({
    connector: {
      id: "cloudflare",
      name: "Cloudflare",
      description: "Manages DNS records and tunnels for your deployed services",
      category: "infrastructure",
      icon: "cloudflare",
    },
    app_configured: false,
  });

const registryConnectors = (): ConnectorStatus[] => [
  connectorEntry(),
  connectorEntry({
    connector: {
      id: "cloudflare",
      name: "Cloudflare",
      description: "Manages DNS records and tunnels for your deployed services",
      category: "infrastructure",
      icon: "cloudflare",
    },
    available: false,
  }),
];

const renderSection = (initialEntries = ["/settings"]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={initialEntries}>
        <ConnectorsSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("ConnectorsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    toastMocks.success.mockClear();
    toastMocks.error.mockClear();
  });

  it("lands on Not connected when nothing is connected, showing every registry connector", async () => {
    mocks.get.mockResolvedValue({ data: registryConnectors() });
    renderSection();

    expect(await screen.findByText("GitHub")).toBeInTheDocument();
    expect(screen.getByText("Cloudflare")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /not connected/i })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("button", { name: /^connect$/i })).toBeInTheDocument();
  });

  it("shows a disabled Coming soon button in place of Connect for a not-yet-available connector", async () => {
    mocks.get.mockResolvedValue({ data: registryConnectors() });
    renderSection();

    expect(await screen.findByRole("button", { name: /coming soon/i })).toBeDisabled();
    expect(screen.getByText("Cloudflare").closest("li")).toHaveTextContent("Coming soon");
  });

  it("lands on Connected and shows the pill and Disconnect button for a configured connector", async () => {
    mocks.get.mockResolvedValue({
      data: [connectorEntry({ status: { configured: true, connected_by: "u1" } })],
    });
    renderSection();

    expect(await screen.findByText("Connected", { selector: "span" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /^connected$/i })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("button", { name: /^disconnect$/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^connect$/i })).not.toBeInTheDocument();
  });

  it("filters the list with the Connected / Not connected tabs", async () => {
    mocks.get.mockResolvedValue({
      data: [
        connectorEntry({ status: { configured: true } }),
        ...registryConnectors().slice(1),
      ],
    });
    const user = userEvent.setup();
    renderSection();

    await screen.findByText("GitHub");
    expect(screen.queryByText("Cloudflare")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /not connected/i }));
    expect(await screen.findByText("Cloudflare")).toBeInTheDocument();
    expect(screen.queryByText("GitHub")).not.toBeInTheDocument();
  });

  it("calls the start-oauth mutation and navigates on Connect", async () => {
    mocks.get.mockResolvedValue({ data: registryConnectors() });
    mocks.post.mockResolvedValue({ data: undefined });
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/connectors/github/oauth/start") {
        return Promise.resolve({ data: { url: "https://github.com/login/oauth/authorize?client_id=x" } });
      }
      return Promise.resolve({ data: registryConnectors() });
    });
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });

    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^connect$/i }));

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/connectors/github/oauth/start"));
    await vi.waitFor(() =>
      expect(assign).toHaveBeenCalledWith("https://github.com/login/oauth/authorize?client_id=x"),
    );

    vi.unstubAllGlobals();
  });

  it("offers an owner Set up app in place of Connect while the OAuth app is unregistered, then saves it", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/auth/me") return Promise.resolve({ data: owner });
      return Promise.resolve({ data: [unregisteredCloudflare()] });
    });
    mocks.put.mockResolvedValue({ data: { configured: true, client_id: "cf-id" } });
    renderSection();

    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Set up app" }));
    expect(screen.queryByRole("button", { name: "Connect" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/app slug/i)).not.toBeInTheDocument();
    await user.type(screen.getByLabelText(/client id/i), "cf-id");
    await user.type(screen.getByLabelText(/client secret/i), "cf-secret");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/connectors/cloudflare/app-config", {
      client_id: "cf-id",
      client_secret: "cf-secret",
      base_url: "",
      app_slug: "",
    });
  });

  it("shows a disabled Connect to non-owners while the OAuth app is unregistered", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/auth/me") return Promise.resolve({ data: member });
      return Promise.resolve({ data: [unregisteredCloudflare()] });
    });
    renderSection();

    expect(await screen.findByRole("button", { name: "Connect" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Set up app" })).not.toBeInTheDocument();
  });

  it("toasts the callback's error param and strips it from the URL", async () => {
    mocks.get.mockResolvedValue({ data: registryConnectors() });
    const { toast } = await import("sonner");
    renderSection(["/settings?connector=github&error=provider%20exploded"]);

    // One-shot effect can fire before the connector list loads, so the name
    // falls back to the id, same as the connected toast.
    await vi.waitFor(() => expect(toast.error).toHaveBeenCalledWith("github: provider exploded"));
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("renders a field form (not the OAuth Connect button) for a manual-credential connector", async () => {
    mocks.get.mockResolvedValue({
      data: [
        connectorEntry({
          connector: {
            id: "livekit",
            name: "LiveKit",
            description: "Powers voice channels with screen share and camera",
            category: "communication",
            icon: "livekit",
            manual: [
              { key: "ws_url", label: "WebSocket URL", secret: false },
              { key: "api_key", label: "API key", secret: false },
              { key: "api_secret", label: "API secret", secret: true },
            ],
          },
        }),
      ],
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^connect$/i }));

    expect(await screen.findByLabelText(/websocket url/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/api secret/i)).toHaveAttribute("type", "password");
    // The OAuth start route must never fire for a manual connector.
    expect(mocks.get).not.toHaveBeenCalledWith("/api/connectors/livekit/oauth/start");
  });

  it("submits the manual credential form and shows Connected on success", async () => {
    mocks.get.mockResolvedValue({
      data: [
        connectorEntry({
          connector: {
            id: "livekit",
            name: "LiveKit",
            description: "Powers voice channels with screen share and camera",
            category: "communication",
            icon: "livekit",
            manual: [
              { key: "ws_url", label: "WebSocket URL", secret: false },
              { key: "api_key", label: "API key", secret: false },
              { key: "api_secret", label: "API secret", secret: true },
            ],
          },
        }),
      ],
    });
    mocks.post.mockResolvedValue({ data: { configured: true, connected_by: "u1" } });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^connect$/i }));
    const dialog = await screen.findByRole("dialog");
    await user.type(within(dialog).getByLabelText(/websocket url/i), "wss://lk.example.com");
    await user.type(within(dialog).getByLabelText(/^api key$/i), "key1");
    await user.type(within(dialog).getByLabelText(/api secret/i), "secret1");
    const fields = { ws_url: "wss://lk.example.com", api_key: "key1", api_secret: "secret1" };

    // Beat one: Verify only calls the provider check and lights up green; nothing is stored yet.
    expect(within(dialog).queryByRole("button", { name: /^confirm$/i })).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: /^verify$/i }));
    expect(await within(dialog).findByText(/accepted these credentials/i)).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/connectors/livekit/manual/verify", fields);
    expect(mocks.post).not.toHaveBeenCalledWith("/api/connectors/livekit/manual", fields);

    // Beat two: Confirm stores and closes.
    await user.click(within(dialog).getByRole("button", { name: /^confirm$/i }));
    await vi.waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/connectors/livekit/manual", fields));
    await vi.waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("verifies each named permission in parallel and ticks rows as they land; a failed row blocks Confirm", async () => {
    mocks.get.mockResolvedValue({
      data: [
        connectorEntry({
          connector: {
            id: "cloudflare",
            name: "Cloudflare",
            description: "Manages DNS records and tunnels for your deployed services",
            category: "infrastructure",
            icon: "cloudflare",
            manual: [{ key: "api_token", label: "API token", secret: true }],
            checks: [
              { key: "token", label: "Token is active" },
              { key: "zone_read", label: "Zone → Zone: Read" },
              { key: "dns_edit", label: "Zone → DNS: Edit" },
              { key: "tunnel_edit", label: "Account → Cloudflare Tunnel: Edit", why: "Creates the tunnel." },
            ],
          },
        }),
      ],
    });
    mocks.post.mockImplementation(async (_url: string, _body: unknown, config?: { params?: { check?: string } }) => {
      if (config?.params?.check === "tunnel_edit") throw new Error("missing");
      return { data: undefined };
    });
    mocks.errorMessage.mockReturnValue("the token is missing Account → Cloudflare Tunnel: Edit");
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^connect$/i }));
    const dialog = await screen.findByRole("dialog");
    // The permissions are listed with their reasons before anything is typed, so the token can be created right.
    expect(within(dialog).getAllByRole("status")).toHaveLength(4);
    expect(within(dialog).getByText("Creates the tunnel.")).toBeInTheDocument();
    expect(within(dialog).getAllByRole("status").every((r) => r.getAttribute("data-state") === "idle")).toBe(true);
    await user.type(within(dialog).getByLabelText(/api token/i), "tok-1");
    await user.click(within(dialog).getByRole("button", { name: /^verify$/i }));

    const rows = await within(dialog).findAllByRole("status");
    expect(rows).toHaveLength(4);
    await vi.waitFor(() => expect(rows.filter((r) => r.getAttribute("data-state") === "pending")).toHaveLength(0));
    for (const key of ["token", "zone_read", "dns_edit", "tunnel_edit"]) {
      expect(mocks.post).toHaveBeenCalledWith(
        "/api/connectors/cloudflare/manual/verify",
        { api_token: "tok-1" },
        { params: { check: key } },
      );
    }
    expect(within(dialog).getByText("Token is active").closest("li")).toHaveAttribute("data-state", "ok");
    expect(within(dialog).getByText("Account → Cloudflare Tunnel: Edit").closest("li")).toHaveAttribute("data-state", "failed");
    expect(within(dialog).getByText(/missing Account → Cloudflare Tunnel: Edit/)).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: /^confirm$/i })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: /^verify$/i })).toBeEnabled();
  });

  it("puts the green light out again when a field changes after Verify", async () => {
    mocks.get.mockResolvedValue({
      data: [
        connectorEntry({
          connector: {
            id: "cloudflare",
            name: "Cloudflare",
            description: "Manages DNS records and tunnels for your deployed services",
            category: "infrastructure",
            icon: "cloudflare",
            manual: [{ key: "api_token", label: "API token", secret: true }],
          },
        }),
      ],
    });
    mocks.post.mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^connect$/i }));
    const dialog = await screen.findByRole("dialog");
    await user.type(within(dialog).getByLabelText(/api token/i), "tok-1");
    await user.click(within(dialog).getByRole("button", { name: /^verify$/i }));
    expect(await within(dialog).findByRole("button", { name: /^confirm$/i })).toBeInTheDocument();

    await user.type(within(dialog).getByLabelText(/api token/i), "x");
    expect(within(dialog).getByRole("button", { name: /^verify$/i })).toBeInTheDocument();
    expect(within(dialog).queryByText(/accepted these credentials/i)).not.toBeInTheDocument();
  });

  it("shows the verify error inline in the manual form on failure", async () => {
    mocks.get.mockResolvedValue({
      data: [
        connectorEntry({
          connector: {
            id: "livekit",
            name: "LiveKit",
            description: "Powers voice channels with screen share and camera",
            category: "communication",
            icon: "livekit",
            manual: [
              { key: "ws_url", label: "WebSocket URL", secret: false },
              { key: "api_key", label: "API key", secret: false },
              { key: "api_secret", label: "API secret", secret: true },
            ],
          },
        }),
      ],
    });
    mocks.post.mockRejectedValue(new Error("failed"));
    mocks.errorMessage.mockReturnValue("bad api key");
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^connect$/i }));
    const dialog = await screen.findByRole("dialog");
    await user.type(within(dialog).getByLabelText(/websocket url/i), "wss://lk.example.com");
    await user.type(within(dialog).getByLabelText(/^api key$/i), "key1");
    await user.type(within(dialog).getByLabelText(/api secret/i), "wrong");
    await user.click(within(dialog).getByRole("button", { name: /^verify$/i }));

    expect(await screen.findByText("bad api key")).toBeInTheDocument();
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: /^confirm$/i })).not.toBeInTheDocument();
  });

  it("calls the disconnect mutation on Disconnect", async () => {
    mocks.get.mockResolvedValue({
      data: [connectorEntry({ status: { configured: true } })],
    });
    mocks.post.mockResolvedValue({ data: { status: "disconnected" } });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^disconnect$/i }));

    await vi.waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/connectors/github/disconnect"));
  });
});
