import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  getMemoryKey,
  getMemoryVersionsKey,
  useCloneMemory,
  useCreateMemory,
  useDeleteMemory,
  useFetchCloneDestinations,
  useFetchMemories,
  useFetchMemoriesByProject,
  useFetchMemory,
  useFetchMemoryVersions,
  useRevertMemory,
  useUpdateMemory,
} from "@/hooks/MemoryHooks";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const memory = {
  id: "mem-1",
  workspace_id: "ws-1",
  project_id: "project-1",
  title: "Deploy quirks",
  when_to_use: "use this if touching deploy config",
  body: "body",
  always_included: false,
  version: 1,
  created_by: "user-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "user-1",
  updated_at: "2026-09-16T12:00:00Z",
};

const memoryVersion = {
  id: "ver-1",
  memory_id: "mem-1",
  version: 1,
  title: "Deploy quirks",
  when_to_use: "use this if touching deploy config",
  body: "body",
  always_included: false,
  author_id: "user-1",
  author_via: "",
  created_at: "2026-09-16T12:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useFetchMemories", () => {
  it("loads the workspace-scoped memory list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [memory] });
    const { result } = renderHook(() => useFetchMemories("ws-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([memory]));
    expect(api.get).toHaveBeenCalledWith("/api/memories", { params: { workspace_id: "ws-1" } });
  });

  it("is disabled without a workspace id", () => {
    const { result } = renderHook(() => useFetchMemories(""), { wrapper });
    expect(result.current.fetchStatus).toBe("idle");
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useFetchMemoriesByProject", () => {
  it("loads the project-scoped memory list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [memory] });
    const { result } = renderHook(() => useFetchMemoriesByProject("project-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([memory]));
    expect(api.get).toHaveBeenCalledWith("/api/memories", { params: { project_id: "project-1" } });
  });

  it("is disabled without a project id", () => {
    const { result } = renderHook(() => useFetchMemoriesByProject(""), { wrapper });
    expect(result.current.fetchStatus).toBe("idle");
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useFetchMemory", () => {
  it("is disabled without an id", () => {
    const { result } = renderHook(() => useFetchMemory(undefined), { wrapper });
    expect(result.current.isPending).toBe(true);
    expect(api.get).not.toHaveBeenCalled();
  });

  it("loads a single memory", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: memory });
    const { result } = renderHook(() => useFetchMemory("mem-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(memory));
    expect(api.get).toHaveBeenCalledWith("/api/memories/mem-1");
  });
});

describe("useCreateMemory", () => {
  it("posts and invalidates the list", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: memory });
    const { result } = renderHook(() => useCreateMemory(), { wrapper });
    await result.current.mutateAsync({
      workspace_id: "ws-1",
      project_id: "project-1",
      title: "Deploy quirks",
      when_to_use: "use this if touching deploy config",
      always_included: false,
    });
    expect(api.post).toHaveBeenCalledWith("/api/memories", {
      workspace_id: "ws-1",
      project_id: "project-1",
      title: "Deploy quirks",
      when_to_use: "use this if touching deploy config",
      always_included: false,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCreateMemory(), { wrapper });
    await result.current
      .mutateAsync({
        workspace_id: "ws-1",
        project_id: "project-1",
        title: "Deploy quirks",
        when_to_use: "",
        always_included: false,
      })
      .catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useUpdateMemory", () => {
  it("puts the fields and invalidates the single-memory key", async () => {
    vi.mocked(api.put).mockResolvedValue({ data: memory });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useUpdateMemory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ id: "mem-1", title: "New title", when_to_use: "x", body: "y", always_included: true });
    expect(api.put).toHaveBeenCalledWith("/api/memories/mem-1", {
      title: "New title",
      when_to_use: "x",
      body: "y",
      always_included: true,
    });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getMemory", "mem-1"] });
  });
});

describe("useDeleteMemory", () => {
  it("deletes and invalidates the list", async () => {
    vi.mocked(api.delete).mockResolvedValue({});
    const { result } = renderHook(() => useDeleteMemory(), { wrapper });
    await result.current.mutateAsync("mem-1");
    expect(api.delete).toHaveBeenCalledWith("/api/memories/mem-1");
  });
});

describe("useFetchMemoryVersions", () => {
  it("loads version history, newest first", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [memoryVersion] });
    const { result } = renderHook(() => useFetchMemoryVersions("mem-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([memoryVersion]));
    expect(api.get).toHaveBeenCalledWith("/api/memories/mem-1/versions");
  });

  it("is disabled without an id", () => {
    const { result } = renderHook(() => useFetchMemoryVersions(undefined), { wrapper });
    expect(result.current.isPending).toBe(true);
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useRevertMemory", () => {
  it("posts the target version and invalidates the memory and its versions", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: memory });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useRevertMemory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ id: "mem-1", version: 1 });
    expect(api.post).toHaveBeenCalledWith("/api/memories/mem-1/revert", { version: 1 });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: [getMemoryKey, "mem-1"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: [getMemoryVersionsKey, "mem-1"] });
  });
});

describe("useCloneMemory", () => {
  it("posts the destination project and invalidates the list", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ...memory, id: "mem-2", project_id: "project-2" } });
    const { result } = renderHook(() => useCloneMemory(), { wrapper });
    const clone = await result.current.mutateAsync({ id: "mem-1", projectId: "project-2", workspaceId: "ws-1" });
    expect(api.post).toHaveBeenCalledWith("/api/memories/mem-1/clone", {
      project_id: "project-2",
      workspace_id: "ws-1",
    });
    expect(clone.id).toBe("mem-2");
  });

  it("clones to workspace scope with an empty project id", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ...memory, id: "mem-3", project_id: "" } });
    const { result } = renderHook(() => useCloneMemory(), { wrapper });
    await result.current.mutateAsync({ id: "mem-1", projectId: "", workspaceId: "ws-1" });
    expect(api.post).toHaveBeenCalledWith("/api/memories/mem-1/clone", { project_id: "", workspace_id: "ws-1" });
  });
});

describe("useFetchCloneDestinations", () => {
  it("groups projects by every workspace the user belongs to", async () => {
    const workspaces = [{ id: "ws-1", name: "Engineering", created_at: "", updated_at: "" }];
    const projects = [{ id: "project-1", name: "Backend", prefix: "BE", position: 0, icon: "", created_at: "", updated_at: "" }];
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces") return { data: workspaces };
      return { data: projects };
    });
    const { result } = renderHook(() => useFetchCloneDestinations(), { wrapper });
    await waitFor(() => expect(result.current.data).toBeDefined());
    expect(result.current.data).toEqual([{ workspace: workspaces[0], projects }]);
  });
});
