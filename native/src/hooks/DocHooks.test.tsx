import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { useFetchDocsByProject } from "@/hooks/DocHooks";

jest.mock("@/api/client", () => ({
  api: { get: jest.fn() },
}));

const docListItem = {
  id: "doc-1",
  project_id: "proj-1",
  title: "Spec",
  version: 3,
  archived: false,
  updated_at: "2026-08-02T00:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  jest.mocked(api.get).mockReset();
});

describe("useFetchDocsByProject", () => {
  test("loads the doc list for a project", async () => {
    jest.mocked(api.get).mockResolvedValue([docListItem]);
    const { result } = await renderHook(() => useFetchDocsByProject("proj-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([docListItem]));
    expect(api.get).toHaveBeenCalledWith("/api/docs?project_id=proj-1");
  });

  test("stays disabled without a project id", async () => {
    const { result } = await renderHook(() => useFetchDocsByProject(undefined), { wrapper });
    expect(result.current.isPending).toBe(true);
    expect(api.get).not.toHaveBeenCalled();
  });
});
