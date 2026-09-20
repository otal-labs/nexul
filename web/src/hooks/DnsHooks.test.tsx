import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  useCreateTunnel,
  useProvisionReverseProxy,
  useProvisionTunnelAgent,
  useRouteTunnelHostname,
} from "@/hooks/DnsHooks";

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

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

describe("DnsHooks tunnel hooks", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.delete.mockReset();
    mocks.toast.success.mockClear();
    mocks.toast.error.mockClear();
  });

  it("useCreateTunnel posts and invalidates the list", async () => {
    mocks.post.mockResolvedValue({ data: { id: "t1", name: "one" } });
    mocks.get.mockResolvedValue({ data: [] });
    const { result } = renderHook(() => useCreateTunnel(), { wrapper });
    result.current.mutate({ name: "one" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels", { name: "one" });
    expect(mocks.toast.success).toHaveBeenCalledWith("Tunnel created");
  });

  it("useCreateTunnel surfaces errors", async () => {
    mocks.post.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("boom");
    const { result } = renderHook(() => useCreateTunnel(), { wrapper });
    result.current.mutate({ name: "one" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(mocks.toast.error).toHaveBeenCalled();
  });

  it("useRouteTunnelHostname posts to the tunnel route endpoint", async () => {
    mocks.post.mockResolvedValue({ data: { id: "t1" } });
    const { result } = renderHook(() => useRouteTunnelHostname(), { wrapper });
    result.current.mutate({
      tunnel_id: "t1",
      hostname: "app.example.com",
      zone_id: "z1",
      zone: "example.com",
      service: "http://localhost:80",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels/t1/route", expect.objectContaining({ hostname: "app.example.com" }));
  });

  it("useProvisionTunnelAgent posts the agent spec", async () => {
    mocks.post.mockResolvedValue({ data: { service_id: "svc-1" } });
    const { result } = renderHook(() => useProvisionTunnelAgent(), { wrapper });
    result.current.mutate({ tunnel_id: "t1", project_id: "p1", target: "10.0.0.1", docker_network: "net" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels/t1/agent", {
      tunnel_id: "t1",
      project_id: "p1",
      target: "10.0.0.1",
      docker_network: "net",
    });
  });

  it("useProvisionReverseProxy posts the proxy spec", async () => {
    mocks.post.mockResolvedValue({ data: { service_id: "svc-1" } });
    const { result } = renderHook(() => useProvisionReverseProxy(), { wrapper });
    result.current.mutate({ project_id: "p1", target: "10.0.0.1", docker_network: "net" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/reverse-proxy", {
      project_id: "p1",
      target: "10.0.0.1",
      docker_network: "net",
    });
  });
});
