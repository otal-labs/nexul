import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { useFetchProjects } from "@/hooks/ProjectHooks";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn() },
}));

const workspace = { id: "ws-1", name: "Acme" };
const project = { id: "proj-1", name: "Nexul", prefix: "NX" };

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
});

describe("useFetchProjects", () => {
  test("resolves the first workspace, then loads its projects", async () => {
    jest.mocked(api.get).mockImplementation((path: string) =>
      path === "/api/workspaces" ? Promise.resolve([workspace]) : Promise.resolve([project]),
    );
    const { result } = await renderHook(() => useFetchProjects(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([project]));
    expect(api.get).toHaveBeenCalledWith("/api/workspaces");
    expect(api.get).toHaveBeenCalledWith("/api/projects?workspace_id=ws-1");
  });

  test("stays disabled while no workspace has loaded", async () => {
    jest.mocked(api.get).mockResolvedValue([]);
    const { result } = await renderHook(() => useFetchProjects(), { wrapper });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces"));
    expect(result.current.isPending).toBe(true);
  });
});
