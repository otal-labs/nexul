import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useFetchServices,
  serviceFollower,
} from "@/hooks/ServiceHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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
  vi.mocked(toast.error).mockClear();
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
});

describe("the service follower", () => {
  const lists = () =>
    seeded([
      [["getServices", "all"], []],
      [["getServices", "proj-1"], []],
      [["getServices", "proj-2"], []],
      [["getServiceDeploys", "svc-1"], [deploy]],
      [["getServiceDeploys", "svc-2"], []],
    ]);
  const keys = [["getServices", "all"], ["getServices", "proj-1"], ["getServices", "proj-2"]];

  it("refetches the stack's project list and the every-project list, and no other project's", async () => {
    const client = lists();
    await followFrame(serviceFollower, "service.updated", { stack: { id: "svc-1", project_id: "proj-1" } }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([true, true, false]);
  });

  it("refetches only the every-project list for the instance's own stack", async () => {
    const client = lists();
    await followFrame(serviceFollower, "service.deleted", { id: "svc-9", name: "gateway" }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([true, false, false]);
  });

  it("refetches the deploy history holding a deploy that moved", async () => {
    const client = lists();
    await followFrame(serviceFollower, "deploy.updated", { id: "d-1", status: "healthy", stack_id: "svc-1" }, client);
    expect([isStale(client, ["getServiceDeploys", "svc-1"]), isStale(client, ["getServiceDeploys", "svc-2"])]).toEqual([true, false]);
  });
});
