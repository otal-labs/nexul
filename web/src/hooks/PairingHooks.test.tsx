import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useHarnessReadiness, pairingFollower } from "@/hooks/PairingHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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
    await waitFor(() => expect(result.current).toEqual({ state: "ready", computerId: "c1", harnessProjectId: "", provider: "", model: "", modelOptions: [] }));
  });

  it("carries the resolved T3 project, provider, model, and model options when ready", async () => {
    mockRoutes({
      resolve: { ok: true, computer_id: "c1", harness_project_id: "t3-home", provider: "claude", model: "sonnet-5", model_options: [{ id: "effort", value: "high" }] },
      presence: { c1: "connected" },
    });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "ready", computerId: "c1", harnessProjectId: "t3-home", provider: "claude", model: "sonnet-5", modelOptions: [{ id: "effort", value: "high" }],
      }),
    );
  });

  it("is offline when the resolved computer is not in the connected presence map", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "offline", message: "T3 Code on your computer is offline." }),
    );
  });

  it("is offline when the resolved computer is only connecting, not connected", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: { c1: "connecting" } });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "offline", message: "T3 Code on your computer is offline." }),
    );
  });

  it("maps unpaired", async () => {
    mockRoutes({ resolve: { ok: false, reason: "unpaired" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({ state: "unpaired", message: "Pair a computer in Settings to run plays." }),
    );
  });

  it("maps expired_token to expired", async () => {
    mockRoutes({ resolve: { ok: false, reason: "expired_token" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "expired",
        message: "Your computer's pairing has expired. Re-pair it in Settings.",
      }),
    );
  });

  it("maps no_default to no_harness_project", async () => {
    mockRoutes({ resolve: { ok: false, reason: "no_default" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "no_harness_project",
        message: "Link this project in Settings → T3 Code Setup → Projects, or set a fallback under Defaults.",
      }),
    );
  });

  it("maps no_default_computer", async () => {
    mockRoutes({ resolve: { ok: false, reason: "no_default_computer" }, presence: {} });
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual({
        state: "no_default_computer",
        message: "Several computers are paired. Pick a default in Settings.",
      }),
    );
  });

  it("passes project_id through to the resolve call when given", async () => {
    mockRoutes({ resolve: { ok: true, computer_id: "c1" }, presence: { c1: "connected" } });
    const { result } = renderHook(() => useHarnessReadiness("proj-1"), { wrapper });
    await waitFor(() => expect(result.current).toEqual({ state: "ready", computerId: "c1", harnessProjectId: "", provider: "", model: "", modelOptions: [] }));
    expect(api.get).toHaveBeenCalledWith("/api/pairing/resolve", { params: { project_id: "proj-1" } });
  });

  it("returns undefined while either query is still loading, never flashing offline", () => {
    vi.mocked(api.get).mockImplementation(() => new Promise(() => {}));
    const { result } = renderHook(() => useHarnessReadiness(), { wrapper });
    expect(result.current).toBeUndefined();
  });
});

describe("the pairing follower", () => {
  it("patches a computer's tunnel checks from the frame, dropping a version the frame no longer carries", async () => {
    const client = seeded([]);
    await followFrame(pairingFollower, "computer.tunnel_status_changed", { computer_id: "c1", user_id: "u1", tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" }, client);
    expect(client.getQueryData(["getTunnelStatus", "c1"])).toEqual({ tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" });
    await followFrame(pairingFollower, "computer.tunnel_status_changed", { computer_id: "c1", user_id: "u1", tunnel: "down", harness_reachable: false }, client);
    expect(client.getQueryData(["getTunnelStatus", "c1"])).toEqual({ tunnel: "down", harness_reachable: false });
  });

  it("refetches the providers and MCP token of the computer a frame names, and no other computer's", async () => {
    const client = seeded([
      [["getHarnessProviders", "c1"], []],
      [["getHarnessProviders", "c2"], []],
      [["getMCPToken", "c1"], null],
      [["getMCPToken", "c2"], null],
    ]);
    await followFrame(pairingFollower, "computer.setup_confirmed", { computer_id: "c2", user_id: "u1", provider: "codex" }, client);
    await followFrame(pairingFollower, "personal_access_token.minted", { token_id: "tok-1", user_id: "u1", name: "Nexul MCP", computer_id: "c2" }, client);
    await followFrame(pairingFollower, "personal_access_token.revoked", { token_id: "tok-2", user_id: "u1", name: "CI" }, client);
    const keys = [["getHarnessProviders", "c1"], ["getHarnessProviders", "c2"], ["getMCPToken", "c1"], ["getMCPToken", "c2"]];
    expect(keys.map((key) => isStale(client, key))).toEqual([false, true, false, true]);
  });

  it("refetches the computer rows and readiness when a computer finishes pairing", async () => {
    const client = seeded([
      [["getComputers"], []],
      [["getHarnessResolve", null], {}],
    ]);
    await followFrame(pairingFollower, "computer.paired", { computer_id: "c1", user_id: "u1" }, client);
    expect([isStale(client, ["getComputers"]), isStale(client, ["getHarnessResolve", null])]).toEqual([true, true]);
  });

  it("refetches the computer rows when a computer's runner connects", async () => {
    const client = seeded([[["getComputers"], []]]);
    await followFrame(pairingFollower, "runner.personal_changed", { computer_id: "c1", user_id: "u1", state: "connected" }, client);
    expect(isStale(client, ["getComputers"])).toBe(true);
  });
});
