import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useCreateService,
  useDeleteService,
  useDeployService,
  useFetchService,
  useFetchServiceDeploys,
  useFetchServices,
  useRollbackService,
  useUpdateService,
} from "@/hooks/ServiceHooks";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const service = { id: "svc-1", name: "api", project_id: "proj-1" };
const deploy = { id: "d-1", service_id: "svc-1", status: "running" };

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("ServiceHooks", () => {
  it("useFetchServices loads by project", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [service] });
    const { result } = renderHook(() => useFetchServices("proj-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([service]));
    expect(api.get).toHaveBeenCalledWith("/api/services", { params: { project_id: "proj-1" } });
  });

  it("useFetchService loads one service", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: service });
    const { result } = renderHook(() => useFetchService("svc-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(service));
    expect(api.get).toHaveBeenCalledWith("/api/services/svc-1");
  });

  it("useFetchServiceDeploys loads deploys", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [deploy] });
    const { result } = renderHook(() => useFetchServiceDeploys("svc-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([deploy]));
    expect(api.get).toHaveBeenCalledWith("/api/services/svc-1/deploys");
  });

  it("useCreateService posts", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: service });
    const { result } = renderHook(() => useCreateService(), { wrapper });
    await result.current.mutateAsync({ name: "api" });
    expect(api.post).toHaveBeenCalledWith("/api/services", { name: "api" });
  });

  it("useUpdateService patches", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: service });
    const { result } = renderHook(() => useUpdateService(), { wrapper });
    await result.current.mutateAsync({ id: "svc-1", payload: { name: "api2" } });
    expect(api.patch).toHaveBeenCalledWith("/api/services/svc-1", { name: "api2" });
  });

  it("useDeleteService deletes", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useDeleteService(), { wrapper });
    await result.current.mutateAsync("svc-1");
    expect(api.delete).toHaveBeenCalledWith("/api/services/svc-1");
  });

  it("useDeployService posts a deploy", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: deploy });
    const { result } = renderHook(() => useDeployService(), { wrapper });
    await result.current.mutateAsync({ serviceId: "svc-1", image: "img:1", ref: "main" });
    expect(api.post).toHaveBeenCalledWith("/api/deploys", { service_id: "svc-1", image: "img:1", ref: "main" });
  });

  it("useRollbackService posts a rollback", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: deploy });
    const { result } = renderHook(() => useRollbackService(), { wrapper });
    await result.current.mutateAsync("svc-1");
    expect(api.post).toHaveBeenCalledWith("/api/services/svc-1/rollback");
  });

  it("hooks toast on error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCreateService(), { wrapper });
    await result.current.mutateAsync({ name: "api" }).catch(() => {});
    await waitFor(() => expect(vi.mocked(api.post)).toHaveBeenCalled());
  });
});
