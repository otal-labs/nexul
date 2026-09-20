import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ServiceHostnameSection } from "@/components/dns/ServiceHostnameSection";
import type { Container } from "@/models/Stack";

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

const containers: Container[] = [
  { id: "svc-api", stack_id: "stack-1", name: "api", declared: {}, status: "healthy", ports: ["8080:8080"] },
];

const renderSection = (list = containers) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ServiceHostnameSection containers={list} />
    </QueryClientProvider>,
  );
};

const cloudflareConnector = (configured: boolean) => [
  {
    connector: { id: "cloudflare", name: "Cloudflare", description: "", category: "infrastructure", icon: "cloudflare" },
    status: { configured },
    available: true,
  },
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

describe("ServiceHostnameSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.delete.mockReset();
    mocks.errorMessage.mockClear();
  });

  it("shows the shared loading display while DNS state loads", async () => {
    mocks.get.mockImplementation(() => new Promise(() => {}));
    renderSection();
    expect(await screen.findByRole("status")).toBeInTheDocument();
  });

  it("hints to connect DNS when no provider is configured", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors") return { data: cloudflareConnector(false) };
      return { data: [] };
    });
    renderSection();
    expect(await screen.findByText(/connect a dns provider/i)).toBeInTheDocument();
  });

  it("shows an empty state when nothing is exposed yet", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors") return { data: cloudflareConnector(true) };
      if (url === "/api/dns/gateways") return { data: [gateway] };
      if (url === "/api/dns/exposures") return { data: [] };
      return { data: [] };
    });
    renderSection();
    expect(await screen.findByText(/not exposed yet/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^expose$/i })).toBeInTheDocument();
  });

  it("lists existing exposures for this stack's containers, by container id, with their gateway kind", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors") return { data: cloudflareConnector(true) };
      if (url === "/api/dns/gateways") return { data: [gateway] };
      if (url === "/api/dns/exposures") {
        return {
          data: [
            {
              id: "exp1",
              gateway_id: "gw1",
              hostname: "api.example.com",
              service_id: "svc-api",
              service: "api",
              port: 8080,
              zone_id: "z1",
              zone: "example.com",
              created_at: "2026-01-01T00:00:00Z",
              updated_at: "2026-01-01T00:00:00Z",
            },
            {
              // Not one of this stack's containers — must not show up.
              id: "exp2",
              gateway_id: "gw1",
              hostname: "other.example.com",
              service_id: "svc-other",
              service: "other",
              port: 9090,
              zone_id: "z1",
              zone: "example.com",
              created_at: "2026-01-01T00:00:00Z",
              updated_at: "2026-01-01T00:00:00Z",
            },
          ],
        };
      }
      return { data: [] };
    });
    mocks.delete.mockResolvedValue({ data: {} });
    renderSection();
    const user = userEvent.setup();

    expect(await screen.findByText("api.example.com")).toBeInTheDocument();
    expect(screen.queryByText("other.example.com")).not.toBeInTheDocument();
    expect(screen.getByText("tunnel")).toBeInTheDocument();
    expect(screen.getByText("api:8080")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /unexpose api.example.com/i }));
    await user.click(screen.getByRole("button", { name: /unexpose/i }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/dns/exposures/exp1");
  });

  it("exposes a container without a client-chosen gateway", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/connectors") return { data: cloudflareConnector(true) };
      if (url === "/api/dns/gateways") return { data: [gateway] };
      if (url === "/api/dns/exposures") return { data: [] };
      if (url === "/api/dns/zones") return { data: [{ id: "z1", name: "example.com" }] };
      return { data: [] };
    });
    mocks.post.mockResolvedValue({ data: {} });
    renderSection();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /^expose$/i }));
    await user.type(screen.getByLabelText(/hostname/i), "api.example.com");
    const zoneTrigger = screen.getByRole("combobox", { name: /zone/i });
    await user.click(zoneTrigger);
    await user.click(await screen.findByRole("option", { name: "example.com" }));
    await user.click(screen.getByRole("button", { name: /^expose$/i }));

    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/exposures",
      expect.objectContaining({ hostname: "api.example.com", service_id: "svc-api", zone_id: "z1", zone: "example.com" }),
    );
  });
});
