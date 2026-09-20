import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useInstanceUpgrade, useRequestInstanceUpgrade } from "@/hooks/InstanceUpgradeHooks";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  errorMessage: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

const idleStatus = {
  version: "v0.2.0-beta-003",
  channel: "beta",
  latest: null,
  update_available: false,
  can_upgrade: false,
  reason: "already on the newest release",
  upgrade: null,
};

describe("useInstanceUpgrade", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("fetches the upgrade status", async () => {
    mocks.get.mockResolvedValue({ data: idleStatus });
    const { result } = renderHook(() => useInstanceUpgrade(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.get).toHaveBeenCalledWith("/api/instance/upgrade");
    expect(result.current.data?.can_upgrade).toBe(false);
  });
});

describe("useRequestInstanceUpgrade", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.toast.success.mockClear();
    mocks.toast.error.mockClear();
  });

  it("posts and toasts on success", async () => {
    mocks.post.mockResolvedValue({
      data: { id: "u-1", from_version: "v1", to_version: "v2", status: "pending", error: "", requested_by: "user-1", created_at: "now", updated_at: "now" },
    });
    const { result } = renderHook(() => useRequestInstanceUpgrade(), { wrapper });
    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/upgrade");
    expect(mocks.toast.success).toHaveBeenCalledWith("Upgrade started");
  });

  it("throws the 409 reason as the error message and toasts it", async () => {
    mocks.post.mockRejectedValue({ response: { status: 409, data: { reason: "instance runner is busy" } } });
    const { result } = renderHook(() => useRequestInstanceUpgrade(), { wrapper });
    result.current.mutate();
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("instance runner is busy");
    expect(mocks.toast.error).toHaveBeenCalled();
  });
});
