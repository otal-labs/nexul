import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useCloneMemory,
  useCreateMemory,
  useDeleteMemory,
  useFetchCloneDestinations,
  useFetchMemories,
  useFetchMemoriesByProject,
  useFetchMemory,
  useUpdateMemory,
  memoryFollower,
} from "@/hooks/MemoryHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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
  footer: false,
  version: 1,
  created_by: "user-1",
  created_at: "2026-09-16T12:00:00Z",
  updated_by: "user-1",
  updated_at: "2026-09-16T12:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(toast.error).mockClear();
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useFetchMemories", () => {
  it("loads the workspace's memory list", async () => {
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
});

describe("useCreateMemory", () => {
  it("posts and invalidates the list", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: memory });
    const client = seeded([[["getMemories", "byProject", "project-1"], []]]);
    const { result } = renderHook(() => useCreateMemory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({
      project_id: "project-1",
      title: "Deploy quirks",
      when_to_use: "use this if touching deploy config",
      always_included: false,
    });
    expect(api.post).toHaveBeenCalledWith("/api/memories", {
      project_id: "project-1",
      title: "Deploy quirks",
      when_to_use: "use this if touching deploy config",
      always_included: false,
    });
    expect(isStale(client, ["getMemories", "byProject", "project-1"])).toBe(true);
  });
});

describe("useCreateMemory failure", () => {
  it("toasts a refusal", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCreateMemory(), { wrapper });
    await result.current.mutateAsync({ project_id: "project-1", title: "Deploy quirks", when_to_use: "", always_included: false }).catch(() => {});
    await waitFor(() => expect(toast.error).toHaveBeenCalled());
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
    const client = seeded([[["getMemories", "byProject", "project-1"], [memory]]]);
    const { result } = renderHook(() => useDeleteMemory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync("mem-1");
    expect(api.delete).toHaveBeenCalledWith("/api/memories/mem-1");
    expect(isStale(client, ["getMemories", "byProject", "project-1"])).toBe(true);
  });
});

describe("useCloneMemory", () => {
  it("posts the destination project and invalidates the list", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { ...memory, id: "mem-2", project_id: "project-2" } });
    const client = seeded([[["getMemories", "byProject", "project-2"], []]]);
    const { result } = renderHook(() => useCloneMemory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    const clone = await result.current.mutateAsync({ id: "mem-1", projectId: "project-2" });
    expect(api.post).toHaveBeenCalledWith("/api/memories/mem-1/clone", { project_id: "project-2" });
    expect(clone.id).toBe("mem-2");
    expect(isStale(client, ["getMemories", "byProject", "project-2"])).toBe(true);
  });
});

describe("useFetchCloneDestinations", () => {
  it("groups projects by every workspace the user belongs to", async () => {
    const workspaces = [{ id: "ws-1", name: "Engineering", slug: "engineering", created_at: "", updated_at: "" }];
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

describe("the memory follower", () => {
  const place = { id: "m-1", workspace_id: "ws-1", project_id: "p-1" };
  const views = () =>
    seeded([
      [["getMemories", "byWorkspace", "ws-1"], []],
      [["getMemories", "byWorkspace", "ws-2"], []],
      [["getMemories", "byProject", "p-1"], []],
      [["getMemories", "byProject", "p-2"], []],
      [["getMemory", "m-1"], {}],
      [["getMemory", "m-2"], {}],
      [["getHistory", "/api/memories/m-1/history"], []],
      [["getInterviewAnswers", "p-1"], []],
    ]);

  it("refetches the edited memory, its history and the two lists holding it, and nothing else", async () => {
    const client = views();
    await followFrame(memoryFollower, "memory.updated", { memory: place, author_id: "u-2" }, client);
    const stale = [["getMemories", "byWorkspace", "ws-1"], ["getMemories", "byProject", "p-1"], ["getMemory", "m-1"], ["getHistory", "/api/memories/m-1/history"]];
    const kept = [["getMemories", "byWorkspace", "ws-2"], ["getMemories", "byProject", "p-2"], ["getMemory", "m-2"], ["getInterviewAnswers", "p-1"]];
    expect(stale.map((key) => isStale(client, key))).toEqual([true, true, true, true]);
    expect(kept.map((key) => isStale(client, key))).toEqual([false, false, false, false]);
  });

  it("refetches the project's memories and answers, and its workspace's list, once its interview run has finished", async () => {
    const client = views();
    const run = { trail_id: "tr-1", play_id: "pl-1", target_type: "interview", target_id: "p-1", activity: null, ended_at: null, last_error: "", project_id: "p-1", workspace_id: "ws-1" };
    await followFrame(memoryFollower, "play.run", { ...run, state: "running" }, client);
    expect(isStale(client, ["getInterviewAnswers", "p-1"])).toBe(false);

    await followFrame(memoryFollower, "play.run", { ...run, state: "done" }, client);
    const keys = [["getMemories", "byProject", "p-1"], ["getInterviewAnswers", "p-1"], ["getMemories", "byWorkspace", "ws-1"], ["getMemories", "byProject", "p-2"], ["getMemories", "byWorkspace", "ws-2"]];
    expect(keys.map((key) => isStale(client, key))).toEqual([true, true, true, false, false]);
  });
});
