import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { getRunnersKey, useCreateRunnerEnrollment, useRemoveRunner, runnerFollower } from "@/hooks/RunnerHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), delete: vi.fn(), toastError: vi.fn(), toastSuccess: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.delete },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : "Something went wrong"),
}));
vi.mock("sonner", () => ({ toast: { error: mocks.toastError, success: mocks.toastSuccess } }));

const runner = {
  id: "r-1",
  name: "alpha",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  running_job: { id: "d-9", kind: "deploy", service: "api" },
  version: "v0.1.4",
};

const wrapperFor = (client: QueryClient) => ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={client}>{children}</QueryClientProvider>
);

describe("RunnerHooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("toasts a failed enrollment", async () => {
    mocks.post.mockRejectedValue(new Error("a runner named alpha is already enrolled"));
    const { result } = renderHook(() => useCreateRunnerEnrollment(), { wrapper: wrapperFor(new QueryClient()) });

    await expect(result.current.mutateAsync({ name: "alpha" })).rejects.toThrow();
    expect(mocks.toastError).toHaveBeenCalledWith("a runner named alpha is already enrolled");
  });

  it("toasts a failed removal", async () => {
    mocks.delete.mockRejectedValue(new Error("forbidden"));
    const { result } = renderHook(() => useRemoveRunner(), { wrapper: wrapperFor(new QueryClient()) });

    result.current.mutate("r-1");
    await waitFor(() => expect(mocks.toastError).toHaveBeenCalledWith("forbidden"));
  });

  it("removes a runner and refreshes the runner list", async () => {
    mocks.delete.mockResolvedValue({});
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useRemoveRunner(), { wrapper: wrapperFor(client) });

    result.current.mutate("r-1");
    await waitFor(() => expect(mocks.toastSuccess).toHaveBeenCalledWith("Runner removed"));
    expect(mocks.delete).toHaveBeenCalledWith("/api/runners/r-1");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getRunnersKey] });
  });
});

describe("the runner follower", () => {
  const busy = () =>
    seeded([
      [["runners"], [{ ...runner, running_job: { id: "d-1", kind: "build" } }]],
      [["runnerQueue"], [{ id: "d-2", kind: "deploy" }]],
    ]);

  it("refetches the runners for a job they do not show yet, and the queue that held it", async () => {
    const client = busy();
    await followFrame(runnerFollower, "deploy.build_started", { id: "d-2", total: 5, log: "step 1" }, client);
    expect([isStale(client, ["runners"]), isStale(client, ["runnerQueue"])]).toEqual([true, true]);
  });

  it.each(["runner.connected", "runner.personal_changed"])("drops a spent enrollment code from the unused ones when %s arrives", async (topic) => {
    const client = seeded([[["runnerEnrollments"], [{ id: "c-1", name: "alpha" }]]]);
    await followFrame(runnerFollower, topic, {}, client);
    expect(isStale(client, ["runnerEnrollments"])).toBe(true);
  });

  it("refetches the runners and the queue once a job ends, since its runner takes the next one", async () => {
    const client = busy();
    await followFrame(runnerFollower, "deploy.status_changed", { id: "d-1", status: "healthy" }, client);
    expect([isStale(client, ["runners"]), isStale(client, ["runnerQueue"])]).toEqual([true, true]);
  });
});
