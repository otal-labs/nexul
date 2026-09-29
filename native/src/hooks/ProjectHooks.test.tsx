import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn() },
}));

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
});


beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("useFetchProjects", () => {
  test("stays disabled while no workspace has loaded", async () => {
    jest.mocked(api.get).mockResolvedValue([]);
    const { result } = await renderHook(() => useFetchProjects(), { wrapper });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(result.current.isPending).toBe(true);
  });
});
