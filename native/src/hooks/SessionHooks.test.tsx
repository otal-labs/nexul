import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { useSignOut } from "@/hooks/SessionHooks";
import { clearPushToken } from "@/push/pushToken";
import { useSessionStore } from "@/stores/sessionStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn(), delete: jest.fn() },
}));

jest.mock("@/push/pushToken", () => ({
  registerPushToken: jest.fn(async () => undefined),
  clearPushToken: jest.fn(async () => undefined),
}));

jest.mock("expo-secure-store", () => {
  const secrets = new Map<string, string>();
  return {
    getItem: (key: string) => secrets.get(key) ?? null,
    setItem: (key: string, value: string) => void secrets.set(key, value),
    deleteItemAsync: async (key: string) => void secrets.delete(key),
  };
});

// sessionStore pulls in the real queryClient, which subscribes to this on mount; unmocked, its subscription lacks .remove() under jest.
jest.mock("expo-network", () => ({
  addNetworkStateListener: () => ({ remove: () => undefined }),
  getNetworkStateAsync: () => Promise.resolve({ isConnected: true }),
}));

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
  jest.mocked(api.delete).mockReset();
  jest.mocked(clearPushToken).mockClear();
  useSessionStore.getState().signIn("https://nexul.example.com", "ses_abc");
});

describe("useSignOut", () => {
  test("clears the local session even when the server delete fails", async () => {
    jest.mocked(api.delete).mockRejectedValue(new Error("offline"));
    const { result } = await renderHook(() => useSignOut(), { wrapper });

    await expect(result.current.mutateAsync()).rejects.toThrow("offline");

    expect(api.delete).toHaveBeenCalledWith("/api/auth/sessions/current");
    expect(useSessionStore.getState().signedIn).toBe(false);
  });

  test("clears the local session on a successful sign-out", async () => {
    jest.mocked(api.delete).mockResolvedValue(undefined);
    const { result } = await renderHook(() => useSignOut(), { wrapper });

    await result.current.mutateAsync();

    expect(useSessionStore.getState().signedIn).toBe(false);
  });

  test("clears the push token while the session can still authenticate, before deleting it", async () => {
    jest.mocked(api.delete).mockResolvedValue(undefined);
    const { result } = await renderHook(() => useSignOut(), { wrapper });

    await result.current.mutateAsync();

    expect(clearPushToken).toHaveBeenCalledWith("https://nexul.example.com", "ses_abc");
    const clearOrder = jest.mocked(clearPushToken).mock.invocationCallOrder[0] ?? Infinity;
    const deleteOrder = jest.mocked(api.delete).mock.invocationCallOrder[0] ?? Infinity;
    expect(clearOrder).toBeLessThan(deleteOrder);
  });
});
