import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useHarnessReadiness } from "@/hooks/PairingHooks";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

// api.get is shared by the resolve call and the presence poll; route by URL.
const mockRoutes = (routes: { resolve: unknown; presence: unknown }) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/pairing/resolve") return { data: routes.resolve };
    if (url === "/api/pairing/presence") return { data: { computers: routes.presence } };
    throw new Error(`unexpected url ${url}`);
  });
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
});

describe("useHarnessReadiness", () => {
  it("is ready when the resolved computer is connected", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: { c1: "connected" } });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() => expect(result.current).toEqual({ state: "ready", computerId: "c1", provider: "", model: "" }));
  });

  it("carries the resolved provider and model when ready", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1", provider: "claude", model: "sonnet-5" }, presence: { c1: "connected" } });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "ready", computerId: "c1", provider: "claude", model: "sonnet-5" }),
    );
  });

  it("is offline when the resolved computer is not in the connected presence map", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "offline", message: "Your harness is offline" }),
    );
  });

  it("is offline when the resolved computer is only connecting, not connected", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: { c1: "connecting" } });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "offline", message: "Your harness is offline" }),
    );
  });

  it("maps unpaired", async () => {
    mockRoutes({ resolve: { ok: false, reason: "unpaired" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "unpaired", message: "Pair a harness in Settings to run plays" }),
    );
  });

  it("maps expired_token to expired", async () => {
    mockRoutes({ resolve: { ok: false, reason: "expired_token" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "expired",
        message: "Your harness pairing has expired, re-pair it in Settings",
      }),
    );
  });

  it("maps no_default to no_harness_project", async () => {
    mockRoutes({ resolve: { ok: false, reason: "no_default" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "no_harness_project",
        message: "Pick a harness project for this project, or set a fallback in Settings",
      }),
    );
  });

  it("maps no_default_computer", async () => {
    mockRoutes({ resolve: { ok: false, reason: "no_default_computer" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "no_default_computer",
        message: "Several harnesses are paired, pick a default in Settings",
      }),
    );
  });

  it("passes project_id through to the resolve call when given", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: { c1: "connected" } });
    const { result } = renderHook(() => useHarnessReadiness("proj-1"), { wrapper });
    await waitFor(() => expect(result.current).toEqual({ state: "ready", computerId: "c1", provider: "", model: "" }));
    expect(api.get).toHaveBeenCalledWith("/api/pairing/resolve", { params: { project_id: "proj-1" } });
  });

  it("returns undefined while either query is still loading, never flashing offline", () => {
    vi.mocked(api.get).mockImplementation(() => new Promise(() => {}));
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    expect(result.current).toBeUndefined();
  });
});
