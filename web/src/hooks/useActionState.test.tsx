import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useActionState } from "@/hooks/useActionState";
import { useSessionStore } from "@/stores/sessionStore";
import { useSetupPassStore } from "@/stores/setupPassStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider>
);

describe("useActionState in first run", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useSessionStore.setState({ token: null, isLoggedIn: false });
    useSetupPassStore.setState({ token: null, expiresAt: null });
  });

  it("is live with the setup pass and never asks /me, whose 401 would drop the pass", () => {
    useSetupPassStore.setState({ token: "pass", expiresAt: null });
    const { result } = renderHook(() => useActionState({ permission: "connectors:write" }), { wrapper });
    expect(result.current).toEqual({ state: "live" });
    expect(mocks.get).not.toHaveBeenCalled();
  });

  it("waits for the account once someone is signed in", () => {
    useSessionStore.setState({ token: "session", isLoggedIn: true });
    mocks.get.mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useActionState({ permission: "connectors:write" }), { wrapper });
    expect(result.current).toEqual({ state: "hidden" });
    expect(mocks.get).toHaveBeenCalledWith("/api/auth/me");
  });
});
