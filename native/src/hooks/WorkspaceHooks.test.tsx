import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";
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
