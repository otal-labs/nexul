import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
  errorMessage: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const projects = [{ id: "p1", name: "Main", created_at: "2026-01-01" }];
const zones = [{ id: "z1", name: "example.com", status: "active" }];
const tunnels = [{ id: "t1", name: "instance", account_id: "acct", status: "active", created_at: "", updated_at: "" }];
const runners = [
  { id: "r1", name: "edge-runner", connected: true, last_seen: "2026-09-03T00:00:00Z", running_job: null, version: "0.1.0" },
];

const gateway = {
  id: "gw1",
  kind: "tunnel",
  docker_network: "nexul",
  service_id: "svc-cf",
  service_name: "cloudflared-instance",
  tunnel_id: "t1",
  zone_id: "z1",
  zone: "example.com",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <GatewaysSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("GatewaysSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.delete.mockReset();
    mocks.errorMessage.mockClear();
    mocks.toast.success.mockClear();
    mocks.toast.error.mockClear();
  });

  it("shows an empty state when there are no gateways", async () => {
    mocks.get.mockImplementation(async () => ({ data: [] }));
    renderSection();
    expect(await screen.findByText(/no gateways yet/i)).toBeInTheDocument();
  });

  it("lists gateways with kind, network, backing service, and zone", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/gateways") return { data: [gateway] };
      return { data: [] };
    });
    renderSection();
    expect(await screen.findByText("nexul")).toBeInTheDocument();
    expect(screen.getByText("tunnel")).toBeInTheDocument();
    expect(screen.getByText("cloudflared-instance")).toBeInTheDocument();
    expect(screen.getByText(/zone example\.com/i)).toBeInTheDocument();
  });

  it("creates a tunnel-kind gateway", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/gateways") return { data: [] };
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/dns/tunnels") return { data: tunnels };
      if (url === "/api/projects") return { data: projects };
      if (url === "/api/runners") return { data: runners };
      return { data: [] };
    });
    mocks.post.mockResolvedValue({ data: gateway });
    renderSection();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /create gateway/i }));
    await user.type(await screen.findByLabelText(/docker network/i), "nexul");
    await pickOption(user, /^Tunnel$/, "instance");
    await pickOption(user, /^Zone$/, "example.com");
    await pickOption(user, /^Project$/, "Main");
    await pickOption(user, "Runs on", "edge-runner");
    await user.click(screen.getByRole("button", { name: /^create gateway$/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/dns/gateways", {
      kind: "tunnel",
      docker_network: "nexul",
      zone_id: "z1",
      zone: "example.com",
      tunnel_id: "t1",
      server_address: "",
      project_id: "p1",
      target: "edge-runner",
    });
  });

  it("switches to the server-address field for a proxy-kind gateway", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/gateways") return { data: [] };
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/dns/tunnels") return { data: tunnels };
      if (url === "/api/projects") return { data: projects };
      if (url === "/api/runners") return { data: runners };
      return { data: [] };
    });
    renderSection();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /create gateway/i }));
    await pickOption(user, /^Kind$/, "Reverse proxy");
    expect(await screen.findByLabelText(/server address/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/^Tunnel$/)).not.toBeInTheDocument();
  });

  it("surfaces the backend's has-exposures rejection when deleting a gateway", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/gateways") return { data: [gateway] };
      return { data: [] };
    });
    mocks.delete.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("gateway gw1 still has 2 exposure(s) — remove them first");
    renderSection();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /delete gateway on nexul/i }));
    await user.click(screen.getByRole("button", { name: /^delete$/i }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/dns/gateways/gw1");
    expect(mocks.toast.error).toHaveBeenCalledWith("gateway gw1 still has 2 exposure(s) — remove them first");
  });
});
