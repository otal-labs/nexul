import { QueryClient, QueryClientProvider, focusManager } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useAreaAccess, useCurrentWorkspaceId, useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn() },
}));

const workspaces = [
  { id: "ws-1", name: "Acme" },
  { id: "ws-2", name: "Beta" },
];

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  jest.mocked(api.get).mockReset().mockResolvedValue(workspaces);
});

describe("useEnsureWorkspaceSelected", () => {
  test("repairs an empty selection to the first workspace the user belongs to", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "" });

    await renderHook(() => useEnsureWorkspaceSelected(), { wrapper });

    await waitFor(() => expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-1"));
  });

  test("repairs a stale selection that no longer belongs to the user", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-gone" });

    await renderHook(() => useEnsureWorkspaceSelected(), { wrapper });

    await waitFor(() => expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-1"));
  });

  test("leaves a valid selection alone", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-2" });

    await renderHook(() => useEnsureWorkspaceSelected(), { wrapper });

    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(useWorkspaceStore.getState().selectedWorkspaceId).toBe("ws-2");
  });
});

// Chat rows, board cards and embeds read these as they mount, and a list mounts rows on every scroll.
describe.each([
  ["useAreaAccess", (): unknown => useAreaAccess(), "/api/workspaces/ws-1/me"],
  ["usePersonLookup", (): unknown => usePersonLookup(useCurrentWorkspaceId()), "/api/workspaces/ws-1/people"],
])("%s, as a list row reads it", (_name, useRowRead, leafPath) => {
  // One client across mounts, as in the app, so rows share its cache.
  const rows = () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    return async () => {
      const row = await renderHook(useRowRead, { wrapper });
      await waitFor(() => expect(api.get).toHaveBeenCalledWith(leafPath));
      await waitFor(() => expect(client.isFetching()).toBe(0));
      return row;
    };
  };

  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    jest.mocked(api.get).mockImplementation((path: string) =>
      Promise.resolve(path === "/api/workspaces" ? workspaces : { people: [], permissions: [] }),
    );
  });

  test("a row mounting again reads the cache without a request", async () => {
    const mountRow = rows();
    await (await mountRow()).unmount();
    const first = jest.mocked(api.get).mock.calls.length;

    for (let i = 0; i < 5; i++) await (await mountRow()).unmount();

    expect(jest.mocked(api.get).mock.calls.length).toBe(first);
  });

  test("returning to the foreground still refreshes it", async () => {
    const row = await rows()();
    const first = jest.mocked(api.get).mock.calls.length;

    await act(async () => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
    });

    await waitFor(() => expect(jest.mocked(api.get).mock.calls.length).toBeGreaterThan(first));
    await row.unmount();
    focusManager.setFocused(undefined);
  });
});
